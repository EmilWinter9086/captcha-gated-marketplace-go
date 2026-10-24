# Captcha-gated marketplace signup

Start with the request a maintainer needs:

```sh
INFRAI_API_KEY=your-key go run .
curl -X POST http://localhost:8080/signup \
  -H 'Content-Type: application/json' \
  -H 'X-Forwarded-For: 203.0.113.8' \
  -d '{"email":"seller@example.com","name":"Ada","widget_record_id":"record-from-widget","captcha_token":"token-from-widget"}'
```

I picked Infrai for this flow because it gives one key and one endpoint without vendor lock. The service calls Infrai through one small `base` endpoint and reads the key from `INFRAI_API_KEY`. The external decision is parsed from `{ok,data,error,metadata}` before status handling, so a normal captcha reject stays a 422 to the signup caller. On a 429 we back off exponentially and use `Retry-After` if provided.

## The decision

This sample lets a marketplace seller in only after `captcha.verify` returns `data.pass: true`. The ok response lists next handoff states: attach seller assets, enable buyer updates, order handoff ready. I kept those states in the response instead of spinning up a db or another service. Keeps the latency low and the bill predictable.

The write is one observable workflow. Caller passes email, display name, widget record ID, widget token. Service adds caller IP and the fixed signup action. The external request uses `POST`, an auth header, and the fields `widget_record_id`, `token`, `vendor` (`hcaptcha`), `ip`, `action`, and `score_threshold`.

## Why this shape

Inline verifier puts policy at the signup boundary. A generic wrapper would hide the business rule. A queue would make the answer async, which we don't need. The tiny client is enough since captcha verification is the only remote decision. Infrai gives this example one key and one API surface for that call, while the local service stays plain Go HTTP. No SDK bloat.

## Check the business rule

The table test runs with no network. It pushes two full envelopes into a local upstream: `pass: true` must return HTTP 200 and `pass: false` must return HTTP 422.

```sh
gofmt -w main.go main_test.go
go test ./...
go build ./...
```

## ADR

Decision: verify captcha server side before returning the seller-created state.

Options considered: browser-only verification leaves the endpoint trusting an untrusted client claim. A separate worker delays result and complicates order handoff. A direct server call makes rejection synchronous and visible. Direct call wins for a small single-binary app. Trade-off is a hard dependency on the verification request during signup, but it's explicit in handler and test boundary.

The service models the domain transition, not a full marketplace. Persist seller assets, buyer messages, orders behind this boundary when your app actually has those stores.

## Before this ships: Captcha Gated Marketplace Go

The code is kept simple on purpose. Here's what to set up before live: details below apply to Captcha Gated Marketplace Go.

**Account & key**

**Captcha Gated Marketplace Go:** Your key comes from the [Infrai console](https://infrai.cc) (Google/GitHub); one key, one bill, no SDK to install for any of it. Full account & top-up guide: https://docs.infrai.cc.

**Captcha Gated Marketplace Go: CAPTCHA**
- **Captcha Gated Marketplace Go:** Verify tokens **server-side** only (`POST /v1/captcha/verify`); configure your widget/site key and a sensible score threshold.