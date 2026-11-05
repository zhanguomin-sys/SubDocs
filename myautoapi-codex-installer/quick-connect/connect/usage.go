package connect

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"time"
)

// KeyState is the outcome of GET /v1/usage for one key.
type KeyState int

const (
	KeyUsable KeyState = iota
	KeyNoBalance
	// KeyRejected is 401 or 403: the key does not exist or is blocked.
	KeyRejected
	// KeyDisabled is a 200 answer for a key the gateway refuses: isValid
	// false, or an expired or used-up key (see keyEnded).
	KeyDisabled
	// KeyTimeout: the server did not answer within usageTimeout (the
	// connection or the answer was too slow).
	KeyTimeout
	// KeyOffline: no connection at all (no network, DNS failure, refused,
	// reset, TLS failure, a proxy or firewall in the way).
	KeyOffline
	// KeyServerError: the server answered, but not with the documented
	// answer (5xx, 404, 429, a redirect or a page that is not JSON).
	KeyServerError
)

// KeyCheck carries the remaining balance in USD when the server sent one,
// and the HTTP status for KeyServerError.
type KeyCheck struct {
	State        KeyState
	Remaining    float64
	HasRemaining bool
	StatusCode   int
}

const (
	usagePath      = "/v1/usage"
	usageTimeout   = 10 * time.Second
	maxUsageAnswer = 4 << 20
)

// NewHTTPClient never follows redirects, so the key only goes to BaseURL.
func NewHTTPClient() *http.Client {
	return &http.Client{
		Timeout: usageTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// CheckKey asks the usage endpoint under baseURL about key. Production code
// always passes BaseURL; tests pass a local fake server.
func CheckKey(ctx context.Context, client *http.Client, baseURL, key string) KeyCheck {
	ctx, cancel := context.WithTimeout(ctx, usageTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+usagePath, nil)
	if err != nil {
		return KeyCheck{State: KeyOffline}
	}
	request.Header.Set("Authorization", "Bearer "+key)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "MyAutoAPI-Quick-Connect/"+Version)
	response, err := client.Do(request)
	if err != nil {
		return networkFailure(err)
	}
	defer response.Body.Close()

	switch response.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return KeyCheck{State: KeyRejected}
	case http.StatusOK:
	default:
		return KeyCheck{State: KeyServerError, StatusCode: response.StatusCode}
	}
	var answer struct {
		IsValid   *bool           `json:"isValid"`
		Status    string          `json:"status"`
		ExpiresAt json.RawMessage `json:"expires_at"`
		Remaining *float64        `json:"remaining"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxUsageAnswer)).Decode(&answer); err != nil {
		if isTimeout(err) {
			return KeyCheck{State: KeyTimeout}
		}
		return KeyCheck{State: KeyServerError, StatusCode: response.StatusCode}
	}
	if answer.IsValid == nil {
		return KeyCheck{State: KeyServerError, StatusCode: response.StatusCode}
	}
	if !*answer.IsValid || keyEnded(answer.Status, answer.ExpiresAt, time.Now()) {
		return KeyCheck{State: KeyDisabled}
	}
	if answer.Remaining == nil {
		// Some subscriptions do not report a balance; that is still usable.
		return KeyCheck{State: KeyUsable}
	}
	check := KeyCheck{State: KeyUsable, Remaining: *answer.Remaining, HasRemaining: true}
	if *answer.Remaining <= 0 {
		check.State = KeyNoBalance
	}
	return check
}

// keyEnded catches keys the gateway refuses although /v1/usage, which skips
// billing checks, still answers isValid true: a quota_limited key whose
// status is expired or quota_exhausted, or whose expires_at has passed while
// the status still says active (the status does not change when the date
// passes).
func keyEnded(status string, expiresAt json.RawMessage, now time.Time) bool {
	if status == "expired" || status == "quota_exhausted" {
		return true
	}
	var text string
	if json.Unmarshal(expiresAt, &text) != nil || text == "" {
		return false
	}
	expires, err := time.Parse(time.RFC3339Nano, text)
	return err == nil && now.After(expires)
}

// networkFailure tells a slow server (timeout) from no connection at all.
func networkFailure(err error) KeyCheck {
	if isTimeout(err) {
		return KeyCheck{State: KeyTimeout}
	}
	return KeyCheck{State: KeyOffline}
}

func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var netError net.Error
	return errors.As(err, &netError) && netError.Timeout()
}
