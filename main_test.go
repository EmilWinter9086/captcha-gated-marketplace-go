package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSignupGateMapsCaptchaDecision(t *testing.T) {
	tests := []struct {
		name, response string
		wantStatus     int
	}{
		{"accepted seller", `{"ok":true,"data":{"pass":true},"error":null,"metadata":{}}`, http.StatusOK},
		{"rejected bot", `{"ok":true,"data":{"pass":false},"error":null,"metadata":{}}`, http.StatusUnprocessableEntity},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(tc.response))
			}))
			defer upstream.Close()
			v := &captchaVerifier{key: "test", client: upstream.Client(), base: upstream.URL}
			server := httptest.NewServer(signupHandler(v))
			defer server.Close()
			res, err := http.Post(server.URL, "application/json", bytesReader(`{"email":"seller@example.com","name":"Seller","widget_record_id":"record","captcha_token":"proof"}`))
			if err != nil {
				t.Fatal(err)
			}
			if res.StatusCode != tc.wantStatus {
				t.Fatalf("status = %d, want %d", res.StatusCode, tc.wantStatus)
			}
		})
	}
}

type stringReader string

func (s stringReader) Read(p []byte) (int, error) {
	if len(s) == 0 {
		return 0, io.EOF
	}
	n := copy(p, s)
	return n, nil
}
func bytesReader(s string) *stringReader { r := stringReader(s); return &r }
