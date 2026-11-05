package connect

import (
	"net/url"
	"strings"
	"testing"
)

func TestParseImportLink(t *testing.T) {
	encoded := url.QueryEscape(fixtureKey)
	cases := []struct {
		name    string
		link    string
		problem LinkProblem
		key     string
		display string
	}{
		{"normal", "myautoapi://import?v=1&app=codex&key=" + fixtureKey + "&name=work", LinkOK, fixtureKey, "work"},
		{"extra slash", "myautoapi://import/?v=1&app=codex&key=" + fixtureKey, LinkOK, fixtureKey, ""},
		{"url encoded", "myautoapi://import?v=1&app=codex&key=" + url.QueryEscape("  "+fixtureKey+" ") + "&name=" + url.QueryEscape("我的 Key & 测试"), LinkOK, fixtureKey, "我的 Key & 测试"},
		{"percent encoded key", "myautoapi://import?v=1&app=codex&key=%73%6B%2D" + encoded[3:], LinkOK, fixtureKey, ""},
		{"scheme and host case", "MyAutoAPI://IMPORT?v=1&app=codex&key=" + fixtureKey, LinkOK, fixtureKey, ""},
		{"parameter order", "myautoapi://import?key=" + fixtureKey + "&app=codex&v=1", LinkOK, fixtureKey, ""},
		{"long name is cut", "myautoapi://import?v=1&app=codex&key=" + fixtureKey + "&name=" + url.QueryEscape("一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十一二三四五"), LinkOK, fixtureKey, "一二三四五六七八九十一二三四五六七八九十一二三四五六七八九十一二…"},
		{"control characters dropped", "myautoapi://import?v=1&app=codex&key=" + fixtureKey + "&name=a%0Ab%07c", LinkOK, fixtureKey, "abc"},

		{"missing key", "myautoapi://import?v=1&app=codex", LinkIncomplete, "", ""},
		{"empty key", "myautoapi://import?v=1&app=codex&key=", LinkIncomplete, "", ""},
		{"bad key", "myautoapi://import?v=1&app=codex&key=pk-abc", LinkIncomplete, "", ""},
		{"key with space", "myautoapi://import?v=1&app=codex&key=sk-abc+def", LinkIncomplete, "", ""},
		{"missing app", "myautoapi://import?v=1&key=" + fixtureKey, LinkIncomplete, "", ""},
		{"missing version", "myautoapi://import?app=codex&key=" + fixtureKey, LinkIncomplete, "", ""},
		{"version not a number", "myautoapi://import?v=one&app=codex&key=" + fixtureKey, LinkIncomplete, "", ""},
		{"version zero", "myautoapi://import?v=0&app=codex&key=" + fixtureKey, LinkIncomplete, "", ""},
		{"address parameter", "myautoapi://import?v=1&app=codex&key=" + fixtureKey + "&base_url=https://evil.example", LinkIncomplete, "", ""},
		{"unknown parameter", "myautoapi://import?v=1&app=codex&key=" + fixtureKey + "&host=evil.example", LinkIncomplete, "", ""},
		{"duplicate key", "myautoapi://import?v=1&app=codex&key=" + fixtureKey + "&key=sk-other", LinkIncomplete, "", ""},
		{"wrong action", "myautoapi://export?v=1&app=codex&key=" + fixtureKey, LinkIncomplete, "", ""},
		{"wrong scheme", "https://import?v=1&app=codex&key=" + fixtureKey, LinkIncomplete, "", ""},
		{"path after host", "myautoapi://import/codex?v=1&app=codex&key=" + fixtureKey, LinkIncomplete, "", ""},
		{"user info", "myautoapi://user@import?v=1&app=codex&key=" + fixtureKey, LinkIncomplete, "", ""},
		{"bad escape", "myautoapi://import?v=1&app=codex&key=%zz", LinkIncomplete, "", ""},
		{"empty", "", LinkIncomplete, "", ""},

		{"version too new", "myautoapi://import?v=2&app=codex&key=" + fixtureKey, LinkUnsupported, "", ""},
		{"version too new with new fields", "myautoapi://import?v=3&app=claude&token=x", LinkUnsupported, "", ""},
		{"claude", "myautoapi://import?v=1&app=claude&key=" + fixtureKey + "&name=work", LinkOK, fixtureKey, "work"},
		{"other app", "myautoapi://import?v=1&app=gemini&key=" + fixtureKey, LinkUnsupported, "", ""},
		{"claude case", "myautoapi://import?v=1&app=Claude&key=" + fixtureKey, LinkUnsupported, "", ""},
		{"app case", "myautoapi://import?v=1&app=Codex&key=" + fixtureKey, LinkUnsupported, "", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			link, problem := ParseImportLink(c.link)
			wantApp := ""
			if c.problem == LinkOK {
				wantApp = "codex"
				if strings.Contains(c.link, "app=claude") {
					wantApp = "claude"
				}
			}
			if problem != c.problem || link.Key != c.key || link.Name != c.display || link.App != wantApp {
				t.Fatalf("ParseImportLink(%q) = %+v, %v; want key %q name %q problem %v", c.link, link, problem, c.key, c.display, c.problem)
			}
		})
	}
}

func TestFindImportLink(t *testing.T) {
	link := "myautoapi://import?v=1&app=codex&key=" + fixtureKey
	if got := FindImportLink([]string{"--flag", "  " + link + " "}); got != link {
		t.Fatalf("got %q", got)
	}
	if got := FindImportLink([]string{"MYAUTOAPI:" + "//import"}); got != "MYAUTOAPI://import" {
		t.Fatalf("scheme case: got %q", got)
	}
	for _, arguments := range [][]string{nil, {}, {"myautoapi"}, {"https://myautoapi.com"}, {"--myautoapi://x"}} {
		if got := FindImportLink(arguments); got != "" {
			t.Fatalf("FindImportLink(%q) = %q", arguments, got)
		}
	}
}

func TestKeyFormatAndMask(t *testing.T) {
	for raw, want := range map[string]bool{
		fixtureKey:               true,
		"  " + fixtureKey + "\n": true,
		"sk-a":                   true,
		"sk-":                    false,
		"":                       false,
		"SK-abc":                 false,
		"pk-abc":                 false,
		"sk-abc def":             false,
		"sk-abc\"x":              false,
		"sk-abc\\x":              false,
		"sk-中文":                  false,
	} {
		if _, ok := NormalizeKey(raw); ok != want {
			t.Errorf("NormalizeKey(%q) ok = %v, want %v", raw, ok, want)
		}
	}
	for key, want := range map[string]string{
		"sk-1a2b3c4d5e6f7a8b9c0dz8y7x6w5": "sk-1a2b3c4d5****z8y7x6w5",
		"sk-1a2b3c4d5ez8y7x6w5":           "sk-****x6w5",
		"sk-12345":                        "sk-****2345",
		"sk-1":                            "sk-****",
		fixtureKey:                        "sk-quickconn****bcdefXYZ",
	} {
		if got := MaskKey(key); got != want {
			t.Errorf("MaskKey(%q) = %q, want %q", key, got, want)
		}
	}
	// The example from the website contract: 12 characters, ****, 8 characters.
	generated := "sk-1a2b3c4d5" + "0000000000000000000000000000000000000000000000000000000" + "z8y7x6w5"
	if got := MaskKey(generated); got != "sk-1a2b3c4d5****z8y7x6w5" {
		t.Fatalf("generated key mask = %q", got)
	}
}
