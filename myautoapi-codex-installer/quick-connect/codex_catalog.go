package main

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	gpt62Astra          = "gpt-6.2-astra"
	gpt62CatalogName    = "myautoapi-gpt-6.2-models.json"
	domesticCatalogName = "myautoapi-models.json"
)

func bundledOfficialModels() ([]byte, error) {
	if embeddedPackages != nil {
		if data, err := fs.ReadFile(embeddedPackages, "packages/official-models.json"); err == nil {
			return data, nil
		}
	}
	return nil, errors.New("未找到内置 GPT 模型目录，请重新下载接入助手")
}

type catalogChange struct {
	path      string
	old, next []byte
	exists    bool
	active    bool
}

func prepareCodexConnection(configPath string, old []byte, official []byte) ([]byte, catalogChange, error) {
	var catalog catalogChange
	next, err := codexConfig(old, func(config map[string]any) error {
		var err error
		catalog, err = prepareGPT62Catalog(filepath.Dir(configPath), config, official)
		return err
	})
	return next, catalog, err
}

type codexModelCatalog struct {
	fields map[string]json.RawMessage
	models []json.RawMessage
	index  map[string]int
}

func readCodexModelCatalog(data []byte) (codexModelCatalog, error) {
	fields, err := jsonObject(data)
	catalog := codexModelCatalog{fields: fields, index: make(map[string]int)}
	if err != nil || json.Unmarshal(fields["models"], &catalog.models) != nil || catalog.models == nil {
		return codexModelCatalog{}, errors.New("模型目录必须包含有效的 models 列表")
	}
	for i, model := range catalog.models {
		var entry struct {
			Slug string `json:"slug"`
		}
		if json.Unmarshal(model, &entry) != nil || entry.Slug == "" {
			return codexModelCatalog{}, errors.New("模型目录有缺少 ID 的条目")
		}
		if _, duplicate := catalog.index[entry.Slug]; duplicate {
			return codexModelCatalog{}, errors.New("模型目录含有重复 ID")
		}
		catalog.index[entry.Slug] = i
	}
	return catalog, nil
}

func (c *codexModelCatalog) addGPT62(official codexModelCatalog) (bool, error) {
	changed := false
	for _, slug := range []string{gpt62Model, gpt62Astra} {
		officialIndex, exists := official.index[slug]
		if !exists || !usableGPT62Model(official.models[officialIndex]) {
			return false, errors.New("内置模型目录缺少支持 High 的 GPT-6.2，请重新下载接入助手")
		}
		if i, present := c.index[slug]; present {
			if usableGPT62Model(c.models[i]) {
				continue
			}
			// Refresh old definitions that would hide the model or make Codex fall
			// back to its catalog default instead of the configured effort.
			c.models[i] = official.models[officialIndex]
		} else {
			c.index[slug] = len(c.models)
			c.models = append(c.models, official.models[officialIndex])
		}
		changed = true
	}
	return changed, nil
}

func usableGPT62Model(data json.RawMessage) bool {
	var model struct {
		Visibility string `json:"visibility"`
		Supported  bool   `json:"supported_in_api"`
		Efforts    []struct {
			Effort string `json:"effort"`
		} `json:"supported_reasoning_levels"`
	}
	if json.Unmarshal(data, &model) != nil || model.Visibility != "list" || !model.Supported {
		return false
	}
	for _, level := range model.Efforts {
		if level.Effort == gpt62Effort {
			return true
		}
	}
	return false
}

func resolveCodexCatalogPath(root, value string) (string, error) {
	if strings.HasPrefix(value, "~/") || strings.HasPrefix(value, "~\\") {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		value = filepath.Join(userHome, value[2:])
	}
	if !filepath.IsAbs(value) {
		value = filepath.Join(root, value)
	}
	return filepath.Abs(value)
}

func sameCatalogPath(left, right string) bool {
	return left == right || runtime.GOOS == "windows" && strings.EqualFold(left, right)
}

func prepareGPT62Catalog(root string, config map[string]any, official []byte) (catalogChange, error) {
	baseline, err := readCodexModelCatalog(official)
	if err != nil {
		return catalogChange{}, errors.New("内置 GPT 模型目录无效，请重新下载接入助手")
	}

	root, err = filepath.Abs(root)
	if err != nil {
		return catalogChange{}, errors.New("无法找到 Codex 配置目录")
	}
	var source string
	var original []byte
	var sourceExists bool
	if value, ok := config["model_catalog_json"]; ok {
		path, valid := value.(string)
		if !valid || strings.TrimSpace(path) == "" {
			return catalogChange{}, errors.New("Codex 配置的模型目录路径无效")
		}
		source, err = resolveCodexCatalogPath(root, path)
		if err != nil {
			return catalogChange{}, errors.New("无法解析现有 Codex 模型目录路径")
		}
		original, sourceExists, err = readConfig(source)
		if err != nil {
			return catalogChange{}, errors.New("无法读取现有 Codex 模型目录，已保留原配置")
		}
	}

	models := baseline
	if sourceExists {
		models, err = readCodexModelCatalog(original)
		if err != nil {
			return catalogChange{}, errors.New("现有 Codex 模型目录格式无效，已保留原配置")
		}
	}
	changed, err := models.addGPT62(baseline)
	if err != nil || (sourceExists && !changed) {
		return catalogChange{}, err
	}

	path := filepath.Join(root, gpt62CatalogName)
	// Keep the domestic tool's own catalog active so its update and uninstall
	// continue working. Read other custom catalogs without editing their files.
	if domestic := filepath.Join(root, domesticCatalogName); sameCatalogPath(source, domestic) {
		path = domestic
	}
	active := sameCatalogPath(source, path)
	previous, exists := original, sourceExists
	if !active {
		previous, exists, err = readConfig(path)
		if err != nil {
			return catalogChange{}, errors.New("无法读取接入助手的 GPT-6.2 模型目录")
		}
		config["model_catalog_json"] = path
	}
	modelList, err := json.Marshal(models.models)
	if err != nil {
		return catalogChange{}, err
	}
	models.fields["models"] = modelList
	updated, err := json.Marshal(models.fields)
	return catalogChange{path: path, old: previous, exists: exists, active: active, next: append(updated, '\n')}, err
}
