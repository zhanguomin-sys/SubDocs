package connect

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Status states shown on the Codex card.
const (
	StateMyAutoAPI = "myautoapi"
	StateNone      = "none"
	StateOther     = "other"
)

// Status is what the Codex card shows. Key is already masked; Name is the
// name the key was imported with, when known.
type Status struct {
	State string `json:"state"`
	Key   string `json:"key"`
	Name  string `json:"name,omitempty"`
	// AppInstalled: the ChatGPT/Codex app seems to be on this computer (see
	// CodexAppDetected); the window offers the download when it is not.
	AppInstalled bool `json:"appInstalled"`
}

// ReadStatus reports MyAutoAPI when the selected provider points at
// api.myautoapi.com, "none" when Codex has no provider and no login, and
// "other" for an official login or any other service.
func ReadStatus(codexHome string) Status {
	status, _ := readStatus(codexHome)
	return status
}

// readStatus also returns the full MyAutoAPI key, for looking up its name.
func readStatus(codexHome string) (Status, string) {
	authData, _ := os.ReadFile(filepath.Join(codexHome, "auth.json"))
	configData, err := os.ReadFile(filepath.Join(codexHome, "config.toml"))
	if err != nil {
		if len(strings.TrimSpace(string(authData))) == 0 {
			return Status{State: StateNone}, ""
		}
		return Status{State: StateOther}, ""
	}
	var config struct {
		ModelProvider  string `toml:"model_provider"`
		ModelProviders map[string]struct {
			BaseURL string `toml:"base_url"`
		} `toml:"model_providers"`
	}
	if _, err := toml.Decode(strings.TrimPrefix(string(configData), "\ufeff"), &config); err != nil {
		return Status{State: StateOther}, ""
	}
	if provider, ok := config.ModelProviders[config.ModelProvider]; ok && isMyAutoAPI(provider.BaseURL) {
		var auth struct {
			Key string `json:"OPENAI_API_KEY"`
		}
		status := Status{State: StateMyAutoAPI}
		if json.Unmarshal(authData, &auth) == nil && auth.Key != "" {
			status.Key = MaskKey(auth.Key)
		}
		return status, auth.Key
	}
	if config.ModelProvider == "" && len(config.ModelProviders) == 0 && len(strings.TrimSpace(string(authData))) == 0 {
		return Status{State: StateNone}, ""
	}
	return Status{State: StateOther}, ""
}

func isMyAutoAPI(address string) bool {
	parsed, err := url.Parse(strings.TrimSpace(address))
	official, _ := url.Parse(BaseURL)
	return err == nil && strings.EqualFold(parsed.Hostname(), official.Hostname())
}
