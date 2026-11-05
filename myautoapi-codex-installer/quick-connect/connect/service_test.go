package connect

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestService(t *testing.T, status int, body string, stop StopResult) (*Service, *int) {
	t.Helper()
	stops := 0
	usage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(usage.Close)
	return &Service{
		CodexHome: filepath.Join(t.TempDir(), ".codex"),
		BaseURL:   usage.URL,
		Client:    NewHTTPClient(),
		Now:       func() time.Time { return fixedTime },
		StopClients: func() StopResult {
			stops++
			return stop
		},
	}, &stops
}

func TestConnectMessages(t *testing.T) {
	running := StopResult{Codex: true}
	stuck := StopResult{Codex: true, Left: true}
	cases := []struct {
		name    string
		status  int
		body    string
		stop    StopResult
		key     string
		tone    string
		message string
		written bool
	}{
		{"success, Codex closed", 200, `{"isValid":true,"remaining":12.345}`, running, fixtureKey, ToneSuccess,
			"已连接 MyAutoAPI ✓ 余额 $12.35。Codex 已关闭，重新打开即可，新建任务，发送“测试”。", true},
		{"success, Codex not running", 200, `{"isValid":true,"remaining":7}`, StopResult{}, " " + fixtureKey + "\n", ToneSuccess,
			"已连接 MyAutoAPI ✓ 余额 $7.00。打开 Codex，新建任务，发送“测试”。", true},
		{"success without balance", 200, `{"isValid":true,"planName":"Pro"}`, StopResult{}, fixtureKey, ToneSuccess,
			"已连接 MyAutoAPI ✓。打开 Codex，新建任务，发送“测试”。", true},
		{"zero balance", 200, `{"isValid":true,"remaining":0}`, StopResult{}, fixtureKey, ToneWarning, MessageNoBalance, true},
		{"zero balance, Codex closed", 200, `{"isValid":true,"remaining":0}`, running, fixtureKey, ToneWarning, MessageNoBalance + MessageClosed, true},
		{"zero balance, Codex stuck", 200, `{"isValid":true,"remaining":-1}`, stuck, fixtureKey, ToneWarning, MessageNoBalance + MessageNotClosed, true},
		{"Codex could not be closed", 200, `{"isValid":true,"remaining":3}`, stuck, fixtureKey, ToneWarning, MessageCloseFailed, true},
		{"rejected 401", 401, `{}`, running, fixtureKey, ToneError, MessageRejected, false},
		{"rejected 403", 403, `{}`, running, fixtureKey, ToneError, MessageRejected, false},
		{"disabled", 200, `{"isValid":false}`, running, fixtureKey, ToneError, MessageDisabled, false},
		{"server error", 503, `{}`, running, fixtureKey, ToneError,
			"MyAutoAPI 服务器暂时出错（HTTP 503），无法确认密钥。Codex 配置没有改动。稍后再点一次，仍不行请进 QQ 群联系客服。", false},
		{"answer not understood", 200, `<html>`, running, fixtureKey, ToneError,
			"MyAutoAPI 服务器暂时出错（返回内容异常），无法确认密钥。Codex 配置没有改动。稍后再点一次，仍不行请进 QQ 群联系客服。", false},
		{"bad format", 200, `{"isValid":true}`, running, "pk-abc", ToneError, MessageBadFormat, false},
		{"masked demo key", 200, `{"isValid":true}`, running, "sk-9f0a66668****c4cfb432", ToneError, MessageBadFormat, false},
		{"empty", 200, `{"isValid":true}`, running, "  ", ToneError, MessageEmptyKey, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			service, stops := newTestService(t, c.status, c.body, c.stop)
			result := service.Connect(context.Background(), c.key, "")
			if result.Tone != c.tone || result.Message != c.message {
				t.Fatalf("got %s %q\nwant %s %q", result.Tone, result.Message, c.tone, c.message)
			}
			if wantAction := c.message == MessageEmptyKey; (result.Action == ActionOpenSite) != wantAction {
				t.Fatalf("action = %q", result.Action)
			}
			if c.written {
				if *stops != 1 || result.Status.State != StateMyAutoAPI || result.Status.Key != MaskKey(fixtureKey) {
					t.Fatalf("after writing: stops %d, status %+v", *stops, result.Status)
				}
			} else {
				if *stops != 0 || result.Status.State != StateNone {
					t.Fatalf("nothing may change: stops %d, status %+v", *stops, result.Status)
				}
				assertOnlyFiles(t, service.CodexHome)
			}
		})
	}
}

func TestNetworkFailureMessages(t *testing.T) {
	closed := httptest.NewServer(http.NotFoundHandler())
	offline := &Service{CodexHome: filepath.Join(t.TempDir(), ".codex"), BaseURL: closed.URL, Client: NewHTTPClient(),
		Now: func() time.Time { return fixedTime }, StopClients: func() StopResult { t.Fatal("must not stop Codex"); return StopResult{} }}
	closed.Close()
	if result := offline.Connect(context.Background(), fixtureKey, ""); result.Message != MessageOffline {
		t.Fatalf("offline: %q", result.Message)
	}

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer slow.Close()
	client := NewHTTPClient()
	client.Timeout = 200 * time.Millisecond
	timeout := &Service{CodexHome: offline.CodexHome, BaseURL: slow.URL, Client: client, Now: offline.Now, StopClients: offline.StopClients}
	if result := timeout.Connect(context.Background(), fixtureKey, ""); result.Message != MessageTimeout {
		t.Fatalf("timeout: %q", result.Message)
	}
	assertOnlyFiles(t, offline.CodexHome)
}

func TestEmptyKeyWhenConnected(t *testing.T) {
	service, _ := newTestService(t, 200, `{"isValid":true,"remaining":1}`, StopResult{})
	service.Connect(context.Background(), fixtureKey, "")
	result := service.Connect(context.Background(), "", "")
	if result.Message != MessageEmptyKeyConnected || result.Action != ActionOpenSite || result.Status.State != StateMyAutoAPI {
		t.Fatalf("%+v", result)
	}
}

func TestImportRemembersName(t *testing.T) {
	service, _ := newTestService(t, 200, `{"isValid":true,"remaining":1}`, StopResult{})
	service.NamesFile = filepath.Join(t.TempDir(), "names", "key-names.json")
	if result := service.Connect(context.Background(), fixtureKey, "工作电脑"); result.Status.Name != "工作电脑" {
		t.Fatalf("after import: %+v", result.Status)
	}
	// Pasting the same key later keeps the name.
	if result := service.Connect(context.Background(), fixtureKey, ""); result.Status.Name != "工作电脑" {
		t.Fatalf("after paste: %+v", result.Status)
	}
	data := readFile(t, service.NamesFile)
	if strings.Contains(data, fixtureKey) || strings.Contains(data, MaskKey(fixtureKey)) {
		t.Fatal("the names file must not hold the key")
	}
	// Another key has no name; the old one keeps its own.
	other := fixtureKey + "x"
	if result := service.Connect(context.Background(), other, ""); result.Status.Name != "" {
		t.Fatalf("other key: %+v", result.Status)
	}
	if result := service.Connect(context.Background(), other, "家里"); result.Status.Name != "家里" {
		t.Fatalf("renamed: %+v", result.Status)
	}
	if lookupName(service.NamesFile, fixtureKey) != "工作电脑" {
		t.Fatal("older name lost")
	}
	for i := 0; i < maxNames+5; i++ {
		if err := rememberName(service.NamesFile, fmt.Sprintf("sk-%d", i), "n"); err != nil {
			t.Fatal(err)
		}
	}
	if got := len(readNames(service.NamesFile)); got != maxNames {
		t.Fatalf("%d names kept", got)
	}
	if restored := service.Restore(); restored.Status.Name != "" {
		t.Fatalf("after restore: %+v", restored.Status)
	}
	// No names file configured: nothing is kept and nothing fails.
	plain, _ := newTestService(t, 200, `{"isValid":true,"remaining":1}`, StopResult{})
	if result := plain.Connect(context.Background(), fixtureKey, "x"); result.Tone != ToneSuccess || result.Status.Name != "" {
		t.Fatalf("%+v", result)
	}
}

func TestConnectKeepsOldConfigWhenWriteFails(t *testing.T) {
	service, stops := newTestService(t, 200, `{"isValid":true,"remaining":1}`, StopResult{Codex: true})
	writeFile(t, filepath.Join(service.CodexHome, "config.toml"), "official config")
	renameFile = func(string, string) error { return &osPathError }
	defer func() { renameFile = defaultRename }()
	result := service.Connect(context.Background(), fixtureKey, "")
	if result.Tone != ToneError || result.Message != "写入 Codex 配置失败：拒绝访问。原配置没有改动，可以重试。" {
		t.Fatalf("got %q", result.Message)
	}
	// Codex is closed before writing, so it is closed even when the write fails.
	if *stops != 1 || readFile(t, filepath.Join(service.CodexHome, "config.toml")) != "official config" {
		t.Fatal("a failed write must keep the config")
	}
}

func TestParseLinkMessages(t *testing.T) {
	service, _ := newTestService(t, 200, `{}`, StopResult{})
	for link, want := range map[string]string{
		"myautoapi://import?v=1&app=codex":                                          MessageLinkIncomplete,
		"myautoapi://import?v=2&app=codex&key=" + fixtureKey:                        MessageLinkUnsupported,
		"myautoapi://import?v=1&app=gemini&key=" + fixtureKey:                       MessageLinkUnsupported,
		"myautoapi://import?v=1&app=codex&key=" + fixtureKey + "&base_url=http://x": MessageLinkIncomplete,
	} {
		result := service.ParseLink(link)
		if result.OK || result.Result.Message != want || result.Result.Tone != ToneError {
			t.Fatalf("%s: %+v", link, result)
		}
	}
	ok := service.ParseLink("myautoapi://import/?v=1&app=codex&key=" + fixtureKey + "&name=%E5%B7%A5%E4%BD%9C")
	if !ok.OK || ok.Key != fixtureKey || ok.Name != "工作" {
		t.Fatalf("%+v", ok)
	}
}

func TestRestoreMessages(t *testing.T) {
	service, stops := newTestService(t, 200, `{"isValid":true,"remaining":1}`, StopResult{ChatGPT: true})
	if result := service.Connect(context.Background(), fixtureKey, ""); result.Tone != ToneSuccess {
		t.Fatal(result.Message)
	}
	result := service.Restore()
	if result.Tone != ToneSuccess || result.Message != MessageRestored || result.Status.State != StateNone || *stops != 2 {
		t.Fatalf("%+v stops %d", result, *stops)
	}

	stuck, _ := newTestService(t, 200, `{}`, StopResult{Codex: true, Left: true})
	if result := stuck.Restore(); result.Tone != ToneWarning || result.Message != MessageRestoredOpen {
		t.Fatalf("%+v", result)
	}

	broken, brokenStops := newTestService(t, 200, `{}`, StopResult{})
	writeFile(t, filepath.Join(broken.CodexHome, "config.toml"), "model = \"x\"\n")
	writeFile(t, filepath.Join(broken.CodexHome, "auth.json"), "{}")
	renameFile = func(string, string) error { return &osPathError }
	result = broken.Restore()
	renameFile = defaultRename
	if result.Tone != ToneError || result.Message != "恢复官方登录失败：拒绝访问。原配置没有改动，可以重试。" || *brokenStops != 0 {
		t.Fatalf("%+v", result)
	}
}

func TestFormatUSD(t *testing.T) {
	for amount, want := range map[float64]string{0: "$0.00", 2: "$2.00", 12.345: "$12.35", 0.004: "$0.00", 1234.5: "$1234.50"} {
		if got := FormatUSD(amount); got != want {
			t.Errorf("FormatUSD(%v) = %q, want %q", amount, got, want)
		}
	}
}
