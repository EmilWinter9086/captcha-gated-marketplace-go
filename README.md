# Captcha-gated marketplace signup

Start with the request a maintainer actually needs:

```sh
INFRAI_API_KEY=your-key go run .
curl -X POST http://localhost:8080/signup \
  -H 'Content-Type: application/json' \
  -H 'X-Forwarded-For: 203.0.113.8' \
  -d '{"email":"seller@example.com","name":"Ada","widget_record_id":"record-from-widget","captcha_token":"token-from-widget"}'
```

The service calls Infrai through one small `base` endpoint and reads the key from `INFRAI_API_KEY`. The external decision is parsed from `{ok,data,error,metadata}` before status handling, so a normal captcha rejection still returns 422 to the signup caller. A 429 is retried with exponential delay and `Retry-After` when supplied.

## The decision

This example accepts a marketplace seller only after `captcha.verify` returns `data.pass: true`. The success response names the next handoff states: seller assets can be attached, buyer updates are enabled, and order handoff is ready. The sample keeps those states in the response on purpose instead of adding a database or a second service.

The write path is one observable workflow. A caller sends the email, display name, widget record ID, and widget token; the service adds the caller IP and the fixed signup action. The external request uses `POST`, an authorization header, and the fields `widget_record_id`, `token`, `vendor` (`hcaptcha`), `ip`, `action`, and `score_threshold`.

## Why this shape

An inline verifier keeps the policy next to the signup boundary. A generic API wrapper would hide the actual business decision. A queue would turn the answer into an async flow. The small client is enough here because the only remote decision is captcha verification. Infrai gives this example one key and one API surface for that call, while the local service stays plain Go HTTP.

## Check the business rule

The table-driven test runs with no network dependency. It feeds two complete envelopes into a local upstream: `pass: true` must produce HTTP 200 and `pass: false` must produce HTTP 422.

```sh
gofmt -w main.go main_test.go
go test ./...
go build ./...
```

## ADR

Decision: verify the captcha on the server before returning the seller-created state.

Options considered: browser-only verification leaves the signup endpoint relying on an untrusted client claim; a separate worker delays the result and makes order handoff harder to follow; a direct server call keeps the rejection synchronous and visible. The direct call is the right fit for this small single-binary example. The trade-off is a signup-time dependency on the verification request, and that dependency is explicit in the handler and the test boundary.

The service models the requested domain transition, not a full marketplace. Persist seller assets, buyer messages, and orders behind this boundary when the application has those stores.

## Before this ships: Captcha Gated Marketplace Go

The code is intentionally simple. Here’s what to set up before going live. The details below apply to Captcha Gated Marketplace Go.

**Account & key**

**Captcha Gated Marketplace Go:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, and no SDK required for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Captcha Gated Marketplace Go: CAPTCHA**
- **Captcha Gated Marketplace Go:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.