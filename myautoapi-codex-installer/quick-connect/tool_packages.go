package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"regexp"
	"strings"
)

const (
	imageTool    = "image"
	modelsTool   = "models"
	imageBase    = "/install/myautoapi-imagines-tool/"
	modelsBase   = "/install/codex-models/"
	imageLimit   = 16 << 20
	catalogLimit = 4 << 20
)

var imageReleaseDir = regexp.MustCompile(`^(v[0-9]{1,4}\.[0-9]{1,4}\.[0-9]{1,4})(?:-upstream-[0-9a-f]{7,40})?$`)

type toolRelease struct {
	Version string `json:"version"`
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
}

func (r toolRelease) valid(kind string) bool {
	if !versionPattern.MatchString(r.Version) || !hashPattern.MatchString(r.SHA256) || r.Size < 1024 {
		return false
	}
	switch kind {
	case imageTool:
		parts := strings.Split(r.Path, "/")
		if r.Size > imageLimit || len(parts) != 3 || parts[0] != "releases" {
			return false
		}
		tag := imageReleaseDir.FindStringSubmatch(parts[1])
		return len(tag) == 2 && tag[1] == "v"+r.Version &&
			parts[2] == "myautoapi-imagines-tool-v"+r.Version+"-windows-amd64.exe"
	case modelsTool:
		return r.Size <= catalogLimit && r.Path == "releases/v"+r.Version+"/catalog.json"
	default:
		return false
	}
}

func (r toolRelease) matches(payload []byte) bool {
	sum := sha256.Sum256(payload)
	return int64(len(payload)) == r.Size && hex.EncodeToString(sum[:]) == r.SHA256
}

type toolBundle struct {
	Image           toolRelease `json:"image"`
	Models          toolRelease `json:"models"`
	ModelCount      int         `json:"model_count"`
	ModelsScriptSHA string      `json:"models_script_sha256"`
	files           fs.FS
}

func readToolBundle(files fs.FS) (toolBundle, error) {
	var b toolBundle
	if files == nil {
		return b, errors.New("此版本没有内置安装包")
	}
	data, err := fs.ReadFile(files, "packages/release.json")
	if err != nil || json.Unmarshal(data, &b) != nil || !b.Image.valid(imageTool) ||
		!b.Models.valid(modelsTool) || !hashPattern.MatchString(b.ModelsScriptSHA) || b.ModelCount < 1 {
		return toolBundle{}, errors.New("内置安装包清单无效，请重新下载助手")
	}
	b.files = files
	return b, nil
}

func (b toolBundle) imageBytes() ([]byte, error) {
	data, err := fs.ReadFile(b.files, "packages/image-tool.exe")
	if err != nil || !looksLikeEXE(data) || !b.Image.matches(data) {
		return nil, errors.New("内置生图程序校验失败，请重新下载助手")
	}
	return data, nil
}

func (b toolBundle) modelScript() ([]byte, error) {
	data, err := fs.ReadFile(b.files, "packages/models.ps1")
	sum := sha256.Sum256(data)
	if err != nil || hex.EncodeToString(sum[:]) != b.ModelsScriptSHA {
		return nil, errors.New("内置模型安装器校验失败，请重新下载助手")
	}
	return data, nil
}

func looksLikeEXE(payload []byte) bool {
	return bytes.HasPrefix(payload, []byte("MZ"))
}

type toolRepository struct {
	client *http.Client
	origin string
}

func (r toolRepository) read(ctx context.Context, path string, limit int64) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.origin+path, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	code := resp.StatusCode
	if code != http.StatusOK {
		return nil, code, fmt.Errorf("官网返回 HTTP %d", code)
	}
	if resp.ContentLength > limit {
		return nil, code, errors.New("官网文件大小不符")
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil || int64(len(data)) > limit {
		return nil, code, errors.New("官网文件不完整或超出大小限制")
	}
	return data, code, nil
}

func (r toolRepository) latest(ctx context.Context, kind string, bundle toolBundle) (toolRelease, error) {
	switch kind {
	case imageTool:
		var manifest struct {
			Version string                 `json:"version"`
			Files   map[string]toolRelease `json:"files"`
		}
		data, _, err := r.read(ctx, imageBase+"latest.json", 32<<10)
		if err != nil {
			return toolRelease{}, err
		}
		if json.Unmarshal(data, &manifest) == nil {
			release := manifest.Files["windows/amd64"]
			release.Version = manifest.Version
			if release.valid(kind) {
				return release, nil
			}
		}
	case modelsTool:
		var release toolRelease
		data, code, err := r.read(ctx, modelsBase+"latest.json", 4096)
		if code == http.StatusNotFound {
			return r.checkLegacyModelCatalog(ctx, bundle)
		}
		if err != nil {
			return toolRelease{}, err
		}
		if json.Unmarshal(data, &release) == nil && release.valid(kind) {
			return release, nil
		}
	default:
		return toolRelease{}, errors.New("请选择受支持的工具")
	}
	return toolRelease{}, errors.New("官网返回的版本清单无效")
}

// Earlier site publishes only the standalone script. Compare its literal catalog
// sections as data; a network-supplied script is never installed or executed.
func (r toolRepository) checkLegacyModelCatalog(ctx context.Context, bundle toolBundle) (toolRelease, error) {
	script, _, err := r.read(ctx, "/install/codex-models.ps1", catalogLimit)
	if err != nil {
		return toolRelease{}, err
	}
	local, err := bundle.modelScript()
	if err != nil {
		return toolRelease{}, err
	}
	if sameLegacyCatalogs(script, local) {
		return bundle.Models, nil
	}
	return toolRelease{}, errors.New("官网还未发布可核对的模型版本清单")
}

func sameLegacyCatalogs(left, right []byte) bool {
	a, ok := readLegacyCatalogs(left)
	b, valid := readLegacyCatalogs(right)
	return ok && valid && a.OfficialVersion == b.OfficialVersion &&
		bytes.Equal(a.Domestic, b.Domestic) && bytes.Equal(a.Official, b.Official)
}

type catalogContents struct {
	Schema          int             `json:"schema"`
	Version         string          `json:"version"`
	OfficialVersion string          `json:"official_version"`
	Domestic        json.RawMessage `json:"domestic"`
	Official        json.RawMessage `json:"official"`
}

func readLegacyCatalogs(script []byte) (catalogContents, bool) {
	script = bytes.ReplaceAll(script, []byte("\r\n"), []byte("\n"))
	extract := func(name string) []byte {
		marker := []byte("\n$" + name + " = ConvertFrom-Json -InputObject @'\n")
		_, after, _ := bytes.Cut(script, marker)
		content, _, found := bytes.Cut(after, []byte("\n'@\n"))
		var compact bytes.Buffer
		if !found || json.Compact(&compact, content) != nil {
			return nil
		}
		return compact.Bytes()
	}
	_, after, _ := bytes.Cut(script, []byte("\n$BundledOfficialVersion = '"))
	value, _, _ := bytes.Cut(after, []byte("'\n"))
	catalog := catalogContents{
		Schema: 1, OfficialVersion: string(value),
		Domestic: extract("DomesticCatalog"), Official: extract("BundledOfficialCatalog"),
	}
	return catalog, versionPattern.MatchString(catalog.OfficialVersion) && len(catalog.Domestic) > 0 && len(catalog.Official) > 0
}

func (r toolRepository) download(ctx context.Context, kind string, release toolRelease) ([]byte, error) {
	if !release.valid(kind) {
		return nil, errors.New("更新地址或校验信息无效")
	}
	base := modelsBase
	if kind == imageTool {
		base = imageBase
	}
	data, _, err := r.read(ctx, base+release.Path, release.Size)
	if err != nil {
		return nil, err
	}
	if !release.matches(data) {
		return nil, errors.New("更新文件 SHA-256 校验失败")
	}
	if kind == imageTool && !looksLikeEXE(data) {
		return nil, errors.New("更新文件不是 Windows 生图程序")
	}
	if kind == modelsTool && !validModelCatalog(data, release.Version) {
		return nil, errors.New("模型目录需要新版接入助手，请先检查助手更新")
	}
	return data, nil
}

var modelID = regexp.MustCompile(`^[a-z0-9][a-z0-9_\.:-]{0,127}$`)

func validModelCatalog(data []byte, expectedVersion string) bool {
	var update catalogContents
	if json.Unmarshal(data, &update) != nil || update.Schema != 1 || update.Version != expectedVersion ||
		!versionPattern.MatchString(update.OfficialVersion) {
		return false
	}
	seen := make(map[string]bool)
	checkModels := func(raw json.RawMessage) bool {
		var list struct {
			Models []struct {
				Slug string `json:"slug"`
			} `json:"models"`
		}
		if json.Unmarshal(raw, &list) != nil || len(list.Models) < 1 || len(list.Models) > 500 {
			return false
		}
		for _, model := range list.Models {
			if !modelID.MatchString(model.Slug) || seen[model.Slug] {
				return false
			}
			seen[model.Slug] = true
		}
		return true
	}
	return checkModels(update.Official) && checkModels(update.Domestic)
}
