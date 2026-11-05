package connect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// fakeUsage is a local stand-in for GET /v1/usage. Nothing reaches the real
// server and no real key is used.
func fakeUsage(t *testing.T, status int, body string) (*httptest.Server, *http.Request) {
	t.Helper()
	seen := &http.Request{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*seen = *r.Clone(context.Background())
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server, seen
}

func TestCheckKeyAnswers(t *testing.T) {
	cases := []struct {
		name      string
		status    int
		body      string
		want      KeyState
		remaining float64
		has       bool
	}{
		{"balance", 200, `{"mode":"unrestricted","isValid":true,"planName":"钱包余额","remaining":12.345,"unit":"USD","balance":12.345}`, KeyUsable, 12.345, true},
		{"zero balance", 200, `{"isValid":true,"remaining":0,"unit":"USD","balance":0}`, KeyNoBalance, 0, true},
		{"negative balance", 200, `{"isValid":true,"remaining":-0.5,"unit":"USD"}`, KeyNoBalance, -0.5, true},
		{"subscription without remaining", 200, `{"mode":"unrestricted","isValid":true,"planName":"Pro","unit":"USD"}`, KeyUsable, 0, false},
		{"remaining null", 200, `{"isValid":true,"remaining":null}`, KeyUsable, 0, false},
		{"quota object does not count", 200, `{"isValid":true,"quota":{"remaining":0},"remaining":3}`, KeyUsable, 3, true},
		{"invalid", 200, `{"mode":"quota_limited","isValid":false,"status":"disabled","remaining":5}`, KeyDisabled, 0, false},
		{"expired status", 200, `{"mode":"quota_limited","isValid":true,"status":"expired"}`, KeyDisabled, 0, false},
		{"quota exhausted status", 200, `{"isValid":true,"mode":"quota_limited","quota":{"limit":5,"remaining":0,"used":5},"remaining":0,"status":"quota_exhausted","unit":"USD"}`, KeyDisabled, 0, false},
		{"expiry date passed while active", 200, `{"days_until_expiry":0,"expires_at":"2001-02-03T04:05:06.789+08:00","isValid":true,"mode":"quota_limited","status":"active"}`, KeyDisabled, 0, false},
		{"expiry date ahead", 200, `{"expires_at":"2999-01-01T00:00:00Z","isValid":true,"mode":"quota_limited","remaining":4,"status":"active","unit":"USD"}`, KeyUsable, 4, true},
		{"expiry null", 200, `{"expires_at":null,"isValid":true,"mode":"quota_limited","status":"active"}`, KeyUsable, 0, false},
		{"unknown expiry format", 200, `{"expires_at":12345,"isValid":true,"status":"active"}`, KeyUsable, 0, false},
		{"unauthorized", 401, `{"error":{"type":"authentication_error","message":"Invalid API key"}}`, KeyRejected, 0, false},
		{"forbidden", 403, `{"error":{"message":"blocked"}}`, KeyRejected, 0, false},
		{"server error", 500, `{"error":"x"}`, KeyServerError, 0, false},
		{"bad gateway", 502, `<html>bad gateway</html>`, KeyServerError, 0, false},
		{"rate limited", 429, `{}`, KeyServerError, 0, false},
		{"not found", 404, `{}`, KeyServerError, 0, false},
		{"not json", 200, `<html>login</html>`, KeyServerError, 0, false},
		{"no isValid", 200, `{"remaining":5}`, KeyServerError, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			server, seen := fakeUsage(t, c.status, c.body)
			got := CheckKey(context.Background(), NewHTTPClient(), server.URL, fixtureKey)
			if c.want == KeyServerError && got.StatusCode != c.status {
				t.Fatalf("status code = %d, want %d", got.StatusCode, c.status)
			}
			if got.State != c.want || got.HasRemaining != c.has || got.Remaining != c.remaining {
				t.Fatalf("CheckKey = %+v, want state %v remaining %v (%v)", got, c.want, c.remaining, c.has)
			}
			if seen.Method != http.MethodGet || seen.URL.Path != "/v1/usage" || seen.Header.Get("Authorization") != "Bearer "+fixtureKey {
				t.Fatalf("request = %s %s %q", seen.Method, seen.URL.Path, seen.Header.Get("Authorization"))
			}
		})
	}
}

func TestCheckKeyNetworkFailures(t *testing.T) {
	closed := httptest.NewServer(http.NotFoundHandler())
	address := closed.URL
	closed.Close()
	if got := CheckKey(context.Background(), NewHTTPClient(), address, fixtureKey); got.State != KeyOffline {
		t.Fatalf("connection refused: %+v", got)
	}

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer slow.Close()
	client := NewHTTPClient()
	client.Timeout = 200 * time.Millisecond
	started := time.Now()
	if got := CheckKey(context.Background(), client, slow.URL, fixtureKey); got.State != KeyTimeout {
		t.Fatalf("timeout: %+v", got)
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("timeout was not applied")
	}

	// Headers arrive in time but the body stalls: still a timeout, not a
	// server error.
	stalled := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"isValid":`))
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	defer stalled.Close()
	if got := CheckKey(context.Background(), client, stalled.URL, fixtureKey); got.State != KeyTimeout {
		t.Fatalf("stalled body: %+v", got)
	}

	// No such host: no connection, not a timeout.
	if got := CheckKey(context.Background(), NewHTTPClient(), "http://myautoapi-test.invalid", fixtureKey); got.State != KeyOffline {
		t.Fatalf("unknown host: %+v", got)
	}
	if NewHTTPClient().Timeout != 10*time.Second {
		t.Fatal("the key check must time out after 10 seconds")
	}
}

func TestCheckKeyDoesNotFollowRedirects(t *testing.T) {
	elsewhere, seen := fakeUsage(t, 200, `{"isValid":true,"remaining":1}`)
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, elsewhere.URL+"/v1/usage", http.StatusFound)
	}))
	defer redirect.Close()
	if got := CheckKey(context.Background(), NewHTTPClient(), redirect.URL, fixtureKey); got.State != KeyServerError || got.StatusCode != http.StatusFound {
		t.Fatalf("redirect: %+v", got)
	}
	if seen.Method != "" {
		t.Fatal("the key was sent to the redirect target")
	}
}
