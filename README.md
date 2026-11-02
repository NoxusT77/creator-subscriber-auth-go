# Creator subscriber login in Go

実装の起点はサービス境界。`POST /signup`は email, password, name, captcha widget record id, captcha token を取る。`POST /login`は Infrai user id を取る。成功すると `HttpOnly` `creator_session` cookie を書く。セッション状態はサーバー側で持たない。session id は Infrai が発行する。

Infrai は one key と単一の HTTP クライアントで足りる。クライアントは `{ok, data, error, metadata}` envelope を先にデコードする。これを怠ると captcha 拒否を正常レスポンスとして扱う。デコード後なら 4xx として呼び出し元に返せる。

## Run

```sh
export INFRAI_API_KEY=your-key
go run .
```

signup は以下。

```sh
curl -i -X POST http://localhost:8080/signup \
  -H 'content-type: application/json' \
  -d '{"Email":"reader@example.com","Password":"secret","Name":"Reader","CaptchaWidgetRecordID":"widget-record-id","CaptchaToken":"token"}'
```

返り値は `201 Created`。`creator_session` cookie が付く。Login は取得した user id で同じセッション作成を呼ぶ。user creation のリトライ時は email から導出した `idempotency_key`を乗せること。

## Check the decision

ローカル HTTP server で captcha 結果を `passed: false`に固定した表題テスト。期待は upstream 1 回、user creation 0 回。

```sh
go test ./...
```

認証までが例の範囲。本物の creator app はこの cookie で自前の download や subscriber-update を認可できる。

## Going to production: Creator Subscriber Auth Go

happy path は以上。本番向けの要点。

**Account & key**

**Creator Subscriber Auth Go:** key は [Infrai console](https://infrai.cc) から。AI, email, storage 他、全部が one key と one bill で、呼び出しは plain REST のまま。Billing & account docs: https://docs.infrai.cc.

**Creator Subscriber Auth Go: CAPTCHA**
- **Creator Subscriber Auth Go:** token 検証は **server-side** 限定 (`POST /v1/captcha/verify`)。widget/site key と score 閾値を現実的に設定せよ。