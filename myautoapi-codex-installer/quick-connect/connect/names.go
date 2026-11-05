package connect

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
)

// Import links carry a display name for the key ("name"). The Codex files have
// no place for it, so the assistant keeps its own small list, keyed by the
// SHA-256 of the key: the key itself is never stored here. The newest names
// come first and the list is capped, so it never grows without bound.
const maxNames = 20

type namedKey struct {
	Hash string `json:"hash"`
	Name string `json:"name"`
}

func keyHash(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func readNames(file string) []namedKey {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	var names []namedKey
	if json.Unmarshal(data, &names) != nil {
		return nil
	}
	return names
}

// lookupName returns the name key was last imported with, or "".
func lookupName(file, key string) string {
	if file == "" {
		return ""
	}
	hash := keyHash(key)
	for _, entry := range readNames(file) {
		if entry.Hash == hash {
			return displayName(entry.Name)
		}
	}
	return ""
}

// rememberName records name for key, replacing an older name for it.
func rememberName(file, key, name string) error {
	if file == "" {
		return nil
	}
	hash := keyHash(key)
	names := []namedKey{{Hash: hash, Name: name}}
	for _, entry := range readNames(file) {
		if entry.Hash != hash && len(names) < maxNames {
			names = append(names, entry)
		}
	}
	data, err := json.Marshal(names)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return err
	}
	temporary := file + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, file)
}

// NamesFile is where the Windows window keeps the names:
// %APPDATA%\MyAutoAPI\QuickConnect\key-names.json.
func NamesFile() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(base, "MyAutoAPI", "QuickConnect", "key-names.json")
}
