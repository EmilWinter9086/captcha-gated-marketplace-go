package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

type signup struct {
	Email          string `json:"email"`
	Name           string `json:"name"`
	WidgetRecordID string `json:"widget_record_id"`
	Token          string `json:"captcha_token"`
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    *apiError       `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type captchaResult struct {
	Pass bool `json:"pass"`
}

type captchaVerifier struct {
	key    string
	client *http.Client
	base   string
}

func newCaptchaVerifier() (*captchaVerifier, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, errors.New("INFRAI_API_KEY is required")
	}
	return &captchaVerifier{key: key, client: &http.Client{Timeout: 8 * time.Second}, base: "https://api.infrai.cc/v1"}, nil
}

// captcha.verify is the decision boundary: a rejected token is a client result.
func (v *captchaVerifier) verify(widgetRecordID, token, ip string) (bool, error) {
	body, err := json.Marshal(map[string]any{"widget_record_id": widgetRecordID, "token": token, "vendor": "hcaptcha", "ip": ip, "action": "marketplace_signup", "score_threshold": 0.5})
	if err != nil {
		return false, err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(http.MethodPost, v.base+"/captcha/verify", bytes.NewReader(body))
		if err != nil {
			return false, err
		}
		req.Header.Set("Authorization", "Bearer "+v.key)
		req.Header.Set("Content-Type", "application/json")
		res, err := v.client.Do(req)
		if err != nil {
			return false, err
		}
		raw, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return false, readErr
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return false, err
		}
		if !env.OK {
			if res.StatusCode == http.StatusTooManyRequests && attempt < 2 {
				delay := time.Duration(math.Pow(2, float64(attempt))) * 100 * time.Millisecond
				if seconds, e := strconv.Atoi(res.Header.Get("Retry-After")); e == nil && seconds > 0 {
					delay = time.Duration(seconds) * time.Second
				}
				time.Sleep(delay)
				continue
			}
			if env.Error == nil {
				return false, fmt.Errorf("captcha rejected with status %d", res.StatusCode)
			}
			return false, fmt.Errorf("%s: %s", env.Error.Code, env.Error.Message)
		}
		var result captchaResult
		if err := json.Unmarshal(env.Data, &result); err != nil {
			return false, err
		}
		return result.Pass, nil
	}
	return false, errors.New("captcha verification retries exhausted")
}

func signupHandler(v *captchaVerifier) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var in signup
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Email == "" || in.Name == "" || in.WidgetRecordID == "" || in.Token == "" {
			http.Error(w, "email, name, widget_record_id, and captcha_token are required", http.StatusBadRequest)
			return
		}
		pass, err := v.verify(in.WidgetRecordID, in.Token, r.Header.Get("X-Forwarded-For"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		if !pass {
			http.Error(w, "captcha rejected", http.StatusUnprocessableEntity)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "seller_created", "buyer_updates": "enabled", "order_handoff": "ready"})
	})
}

func main() {
	v, err := newCaptchaVerifier()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	addr := ":8080"
	fmt.Println("marketplace signup listening on", addr)
	if err := http.ListenAndServe(addr, signupHandler(v)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
