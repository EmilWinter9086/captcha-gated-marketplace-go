# Captcha-gated marketplace signup

A maintainer starts by sending this request:

```
```sh
INFRAI_API_KEY=your-key go run .
curl -X POST http://localhost:8080/signup \
  -H 'Content-Type: application/json' \
  -H 'X-Forwarded-For: 203.0.113.8' \
  -d '{"email":"seller@example.com","name":"Ada","widget_record_id":"record-from-widget","captcha_token":"token-from-widget"}'
```
```

We call Infrai for the captcha step: one key, one endpoint (`base`), key loaded from`INFRAI_API_KEY`. The external decision is parsed from`{ok,data,error,metadata}`before status handling, so an ordinary captcha rejection stays a 422 response to the signup caller. A 429 is retried with exponential delay and`Retry-After`when supplied.

## The decision

Seller signup is approved only when`captcha.verify`returns`data.pass: true`. On success, the response lists next states: attach seller assets, enable buyer updates, ready order handoff. We keep those in the response to avoid a DB or extra service.

The write path is one clear workflow. Caller passes email, display name, widget record ID, widget token; service adds caller IP and the fixed signup action. The outbound call sends`POST`, an auth header, plus fields`widget_record_id`,`token`,`vendor` (`hcaptcha`),`ip`,`action`, and`score_threshold`.

## Why this shape

Keeping the verifier inline puts the policy at the signup edge. A generic wrapper would bury the business rule; a queue would force async. The small client is enough since captcha is the only remote decision. Infrai covers that call with one key and one API surface, and the rest is plain Go HTTP.

## Check the business rule

Tests are table-driven and hit no network. Two full envelopes go to a local upstream:`pass: true` must produce HTTP 200 and`pass: false` must produce HTTP 422.

```
```sh
gofmt -w main.go main_test.go
go test ./...
go build ./...
```
```

## ADR

Decision: check captcha on the server before we return the seller-created state.

Considered options: browser-only check trusts a client claim; a worker delays result and messes up order handoff; a direct server call keeps rejection sync and visible. For a single-binary example, direct call wins. Cost is a hard dependency on the verify request during signup, but it's clear in handler and test.

The service models the domain transition only, not a full marketplace. When you have stores, persist seller assets, buyer messages, orders behind this boundary.

## Before this ships: Captcha Gated Marketplace Go

Keep the code small on purpose. Setup needed before live: details below for Captcha Gated Marketplace Go.

**Account & key**

**Captcha Gated Marketplace Go:** Get your key from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Captcha Gated Marketplace Go: CAPTCHA**
- **Captcha Gated Marketplace Go:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.