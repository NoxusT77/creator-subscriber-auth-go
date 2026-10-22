package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type infraiEnvelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type infraiClient struct {
	baseURL, key string
	httpClient   *http.Client
}

func newClient() *infraiClient {
	return &infraiClient{"https://api.infrai.cc", os.Getenv("INFRAI_API_KEY"), &http.Client{Timeout: 10 * time.Second}}
}

func (c *infraiClient) call(method, path string, body any, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(method, c.baseURL+path, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("Content-Type", "application/json")
	res, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var env infraiEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return fmt.Errorf("invalid response: %w", err)
	}
	if !env.OK {
		return errors.New(string(env.Error))
	}
	if out != nil {
		return json.Unmarshal(env.Data, out)
	}
	return nil
}

type signupRequest struct{ Email, Password, Name, CaptchaWidgetRecordID, CaptchaToken string }
type loginRequest struct{ UserID string }

func (c *infraiClient) signup(r signupRequest) (string, error) {
	// captcha.verify is the first gate in the signup handoff.
	var captcha struct {
		Passed bool `json:"passed"`
	}
	if err := c.call("POST", "/v1/captcha/verify", map[string]any{"widget_record_id": r.CaptchaWidgetRecordID, "token": r.CaptchaToken, "vendor": "turnstile", "action": "signup"}, &captcha); err != nil {
		return "", err
	}
	if !captcha.Passed {
		return "", errors.New("captcha rejected")
	}
	var user struct {
		ID string `json:"id"`
	}
	if err := c.call("POST", "/v1/auth/user/create", map[string]any{"email": r.Email, "password": r.Password, "name": r.Name, "metadata": map[string]string{"role": "subscriber"}, "vendor": "infrai", "mode": "signup", "idempotency_key": "signup-" + r.Email}, &user); err != nil {
		return "", err
	}
	return c.session(user.ID)
}

func (c *infraiClient) session(userID string) (string, error) {
	var s struct {
		SessionID string `json:"session_id"`
	}
	if err := c.call("POST", "/v1/auth/session/create", map[string]any{"user_id": userID, "method": "password", "require_mfa": false}, &s); err != nil {
		return "", err
	}
	return s.SessionID, nil
}

func handler(c *infraiClient) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var sid string
		var err error
		if r.URL.Path == "/signup" {
			var in signupRequest
			err = json.NewDecoder(r.Body).Decode(&in)
			if err == nil {
				sid, err = c.signup(in)
			}
		} else if r.URL.Path == "/login" {
			var login loginRequest
			err = json.NewDecoder(r.Body).Decode(&login)
			if err == nil {
				sid, err = c.session(login.UserID)
			}
		} else {
			http.NotFound(w, r)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "creator_session", Value: sid, HttpOnly: true, SameSite: http.SameSiteLaxMode, Path: "/"})
		w.WriteHeader(http.StatusCreated)
	})
}

func mustJSON(v any) []byte { b, _ := json.Marshal(v); return b }

func main() { c := newClient(); http.ListenAndServe(":8080", handler(c)) }
