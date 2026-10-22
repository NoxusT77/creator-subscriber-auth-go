package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignupCaptchaDecision(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["widget_record_id"] != "widget-1" || body["token"] != "x" {
			t.Fatalf("unexpected captcha request: %#v", body)
		}
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "data": map[string]any{"passed": false}})
	}))
	defer srv.Close()
	c := &infraiClient{baseURL: srv.URL, key: "test", httpClient: srv.Client()}
	_, err := c.signup(signupRequest{Email: "a@example.com", Password: "pw", CaptchaWidgetRecordID: "widget-1", CaptchaToken: "x"})
	if err == nil || calls != 1 {
		t.Fatalf("expected captcha rejection before user creation, err=%v calls=%d", err, calls)
	}
}
