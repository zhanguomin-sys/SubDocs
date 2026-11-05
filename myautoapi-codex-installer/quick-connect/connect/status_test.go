package connect

import (
	"path/filepath"
	"testing"
)

func TestReadStatus(t *testing.T) {
	cases := []struct {
		name   string
		config string // "" means no file
		auth   string
		want   Status
	}{
		{"nothing", "", "", Status{State: StateNone}},
		{"written by the assistant", string(mustRead("testdata/expected-config.toml")), string(mustRead("testdata/expected-auth.json")), Status{State: StateMyAutoAPI, Key: "sk-quickconn****bcdefXYZ"}},
		{"myautoapi with path and no key", "model_provider = \"relay\"\n[model_providers.relay]\nbase_url = \"https://API.myautoapi.com/v1\"\n", "", Status{State: StateMyAutoAPI}},
		{"after restore", "model_reasoning_effort = \"high\"\n\n[features]\ngoals = true\n", "", Status{State: StateNone}},
		{"official login", "[features]\ngoals = true\n", `{"auth_mode":"chatgpt","tokens":{"id_token":"x"}}`, Status{State: StateOther}},
		{"official login without config", "", `{"tokens":{}}`, Status{State: StateOther}},
		{"other relay", "model_provider = \"x\"\n[model_providers.x]\nbase_url = \"https://relay.example.com\"\n", `{"OPENAI_API_KEY":"sk-other"}`, Status{State: StateOther}},
		{"lookalike host", "model_provider = \"x\"\n[model_providers.x]\nbase_url = \"https://api.myautoapi.com.evil.example\"\n", "", Status{State: StateOther}},
		{"provider not selected", "[model_providers.OpenAI]\nbase_url = \"https://api.myautoapi.com\"\n", "", Status{State: StateOther}},
		{"broken toml", "model_provider = \n[[", "", Status{State: StateOther}},
		{"bom", "\ufeff" + string(mustRead("testdata/expected-config.toml")), `{"OPENAI_API_KEY":"sk-1a2b3c4d5e6f7a8b9c0dz8y7x6w5"}`, Status{State: StateMyAutoAPI, Key: "sk-1a2b3c4d5****z8y7x6w5"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			codexHome := filepath.Join(t.TempDir(), ".codex")
			if c.config != "" {
				writeFile(t, filepath.Join(codexHome, "config.toml"), c.config)
			}
			if c.auth != "" {
				writeFile(t, filepath.Join(codexHome, "auth.json"), c.auth)
			}
			if got := ReadStatus(codexHome); got != c.want {
				t.Fatalf("ReadStatus = %+v, want %+v", got, c.want)
			}
		})
	}
}
