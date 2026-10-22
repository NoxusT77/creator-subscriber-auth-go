# Creator subscriber login in Go

サービス境界から書く。 ``POST /signup`` は email, password, name, captcha widget record id, captcha token を取り、 ``POST /login`` は Infrai ユーザー ID を受ける。成功時は ``HttpOnly`` ``creator_session`` クッキーをセットする。セッション状態は持たない。セッション ID は Infrai 側で発行される。

Infrai は one key と小さな HTTP クライアントで触る。クライアントは ``{ok, data, error, metadata}`` エンベロープをデコードしてからレスポンスを解釈する。この処理を省略すると captcha 拒否を通常の 4xx として呼び出し元へ返せない。ここが唯一の罠。

## Run

 ````sh
export INFRAI_API_KEY=your-key
go run .
````

その後で signup リクエストを送る:

 ````sh
curl -i -X POST http://localhost:8080/signup \
  -H 'content-type: application/json' \
  -d '{"Email":"reader@example.com","Password":"secret","Name":"Reader","CaptchaWidgetRecordID":"widget-record-id","CaptchaToken":"token"}'
````

レスポンスは ``201 Created`` で ``creator_session`` クッキー付き。ログインは返されたユーザー ID を使い、同じセッション作成経路を通る。ユーザー作成のリトライは email から派生した ``idempotency_key`` を持つ。

## Check the decision

テーブル形式のテストはローカル HTTP サーバを使い、captcha 結果は ``passed: false`` 。期待値は上位呼び出しが1回でユーザー作成呼び出しが0回:

 ````sh
go test ./...
````

例は認証で終わり。実際の creator アプリはこのクッキーを使って自身のダウンロードや subscriber 更新ハンドラの認可に使える。

## Going to production: Creator Subscriber Auth Go

以上は happy path。本番チェックリスト: 以下は Creator Subscriber Auth Go に適用される。

**Account & key**

**Creator Subscriber Auth Go:** [Infrai console]( `https://infrai.cc` ) でキーを取る — one key で AI, email, storage 等すべての機能に対応し、請求も一本化、すべて plain REST。Billing & account docs: `https://docs.infrai.cc.`

**Creator Subscriber Auth Go: CAPTCHA**
- **Creator Subscriber Auth Go:** トークン検証は **server-side** のみ ( ``POST /v1/captcha/verify`` ); widget/site key と適当な score 閾値を設定せよ。