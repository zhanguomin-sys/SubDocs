package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

const apiModelsURL = "https://api.myautoapi.com/v1/models"

func models(arguments []string, stdout io.Writer) error {
	if len(arguments) != 0 {
		return errors.New("models 不接受参数。")
	}
	root, err := codexHome()
	if err != nil {
		return err
	}
	key, err := loadAPIKey(root)
	if err != nil {
		return err
	}
	return requestModels(stdout, newHTTPClient(), apiModelsURL, key)
}

func requestModels(stdout io.Writer, client httpDoer, endpoint, key string) error {
	request, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return errors.New("无法创建模型列表请求。")
	}
	request.Header.Set("Authorization", "Bearer "+key)
	request.Header.Set("User-Agent", "myautoapi-imagines-tool/"+version())
	response, err := client.Do(request)
	if err != nil {
		return errors.New(redactError("模型探测失败："+err.Error(), key))
	}
	defer response.Body.Close()
	content, err := readLimited(response.Body)
	if err != nil {
		return errors.New(redactError("无法读取模型列表："+err.Error(), key))
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("模型探测返回 HTTP %d。", response.StatusCode)
	}
	var body struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			Name        string `json:"name"`
		} `json:"data"`
	}
	if err := json.Unmarshal(content, &body); err != nil || body.Data == nil {
		return errors.New("模型列表响应不是有效的模型 JSON。")
	}
	result := make([]map[string]any, 0, len(body.Data))
	for _, item := range body.Data {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		name := item.DisplayName
		if name == "" {
			name = item.Name
		}
		if name == "" {
			name = item.ID
		}
		_, supported := imageModelSpecs[item.ID]
		result = append(result, map[string]any{"id": item.ID, "display_name": name, "tool_supported": supported})
	}
	writeResult(stdout, map[string]any{"ok": true, "http_status": response.StatusCode, "models": result, "count": len(result)})
	return nil
}

func paths(arguments []string, stdout io.Writer) error {
	if len(arguments) != 0 {
		return errors.New("paths 不接受参数。")
	}
	desktop, err := desktopDirectory()
	if err != nil {
		return err
	}
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("无法读取当前目录：%w", err)
	}
	root, err := codexHome()
	if err != nil {
		return err
	}
	writeResult(stdout, map[string]any{
		"ok": true, "desktop": desktop, "desktop_images": filepath.Join(desktop, "images"),
		"working_directory": cwd, "project_images": filepath.Join(cwd, "images"),
		"codex_home": root, "tool_path": installedToolPath(buildInstallLayout(root)),
	})
	return nil
}

type diagnosticCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

func doctor(arguments []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("doctor", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var outputDir string
	flags.StringVar(&outputDir, "output-dir", "", "需要检查的图片目录")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return errors.New("doctor 只接受可选的 --output-dir。")
	}
	client := newHTTPClient()
	client.Timeout = 20 * time.Second
	return runDoctor(outputDir, stdout, client)
}

func runDoctor(outputDir string, stdout io.Writer, client httpDoer) error {
	root, err := codexHome()
	if err != nil {
		return err
	}
	layout := buildInstallLayout(root)
	checks := make([]diagnosticCheck, 0)
	add := func(name string, ok bool, detail string) { checks = append(checks, diagnosticCheck{name, ok, detail}) }
	if err := checkManagedPaths(layout); err != nil {
		add("managed_paths", false, err.Error())
	} else {
		state, stateErr := readInstallState(layout)
		add("install_manifest", stateErr == nil && state != nil, "检查工具安装记录，不显示 Key。")
		if stateErr == nil && state != nil {
			managed, hashes, manifestErr := managedPathsFromState(layout, state)
			if manifestErr != nil {
				add("managed_files", false, manifestErr.Error())
			} else {
				for logical, path := range managed {
					info, statErr := os.Lstat(path)
					valid := statErr == nil && info.Mode().IsRegular()
					if valid {
						content, readErr := os.ReadFile(path)
						valid = readErr == nil && sha256Hex(content) == hashes[logical]
					}
					add(logical, valid, path)
				}
			}
		}
		keyInfo, keyErr := os.Lstat(filepath.Join(layout.toolDir, keyFileName))
		add("dedicated_key", keyErr == nil && keyInfo.Mode().IsRegular() && keyInfo.Size() > 0, "只检查专门画图分组 Key 文件是否存在且非空，不读取或显示内容。")
		agents, agentsErr := readAgents(layout, false)
		add("global_rules", agentsErr == nil && strings.Contains(agents, ruleStart) && strings.Contains(agents, ruleEnd), layout.agentsPath)
		policy, policyErr := os.ReadFile(layout.rulesPath)
		add("exec_policy", policyErr == nil && bytes.Equal(policy, []byte(renderExecPolicyRules(layout))), "检查放行规则文件；无法判断当前会话是否已加载。安装后必须完全退出并重新打开 Codex。")
	}
	request, _ := http.NewRequest(http.MethodHead, "https://api.myautoapi.com", nil)
	response, networkErr := client.Do(request)
	if response != nil {
		_ = response.Body.Close()
	}
	networkDetail := "连接 api.myautoapi.com；不发送 Key，不调用付费生图接口。"
	if networkErr != nil {
		networkDetail += " " + redactError(networkErr.Error(), "")
	}
	add("network", networkErr == nil, networkDetail)
	if outputDir != "" {
		outputErr := os.MkdirAll(outputDir, 0o777)
		if outputErr == nil {
			file, createErr := os.CreateTemp(outputDir, ".myautoapi-write-check-*")
			outputErr = createErr
			if createErr == nil {
				_ = file.Close()
				outputErr = os.Remove(file.Name())
			}
		}
		add("output_directory", outputErr == nil, outputDir)
	}
	ok := true
	for _, check := range checks {
		if !check.OK {
			ok = false
		}
	}
	writeResult(stdout, map[string]any{"ok": ok, "checks": checks})
	return nil
}

func recoverOutputs(arguments []string, stdout io.Writer) error {
	flags := flag.NewFlagSet("recover", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var outputDir, name string
	var since int64
	var n int
	flags.StringVar(&outputDir, "output-dir", "", "本次输出目录")
	flags.StringVar(&name, "output-name", "", "本次输出名称")
	flags.Int64Var(&since, "since", 0, "本次调用开始时间，Unix 毫秒")
	flags.IntVar(&n, "n", 1, "本次请求数量")
	if err := flags.Parse(arguments); err != nil || flags.NArg() != 0 {
		return errors.New("recover 参数无效。")
	}
	if outputDir == "" || name == "" || since <= 0 || n < 1 || n > 10 {
		return errors.New("recover 必须指定目录、名称、有效的开始时间及 1 至 10 的请求数量。")
	}
	root, err := filepath.Abs(outputDir)
	if err != nil {
		return errors.New("无法确定本次输出目录。")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("无法读取本次输出目录：%w", err)
	}
	pattern := regexp.MustCompile("^" + regexp.QuoteMeta(sanitizeOutputName(name)) + `(?:-[1-9][0-9]*){0,2}$`)
	type candidate struct {
		path     string
		modified time.Time
	}
	found := make([]candidate, 0)
	for _, entry := range entries {
		if entry.Type()&os.ModeSymlink != 0 || !isKnownImageExtension(filepath.Ext(entry.Name())) {
			continue
		}
		base := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		if !pattern.MatchString(base) {
			continue
		}
		info, statErr := entry.Info()
		if statErr != nil || !info.Mode().IsRegular() || info.ModTime().UnixMilli() < since || info.Size() <= 0 || info.Size() > maxResponseBytes {
			continue
		}
		content, readErr := os.ReadFile(filepath.Join(root, entry.Name()))
		if readErr != nil {
			continue
		}
		if inspectImage(content).format == "unknown" {
			continue
		}
		found = append(found, candidate{filepath.Join(root, entry.Name()), info.ModTime()})
	}
	if len(found) > n {
		return errors.New("匹配到的本次图片超过请求数量，无法安全确认结果；未选择任意最近图片。")
	}
	sort.Slice(found, func(i, j int) bool {
		if found[i].modified.Equal(found[j].modified) {
			return found[i].path < found[j].path
		}
		return found[i].modified.Before(found[j].modified)
	})
	files := make([]string, 0, len(found))
	for _, item := range found {
		files = append(files, item.path)
	}
	summary := outputSummary(files, n)
	var gifs []string
	for _, path := range files {
		if strings.EqualFold(filepath.Ext(path), ".gif") {
			gifs = append(gifs, path)
		}
	}
	if len(gifs) > 0 {
		addGIFSummary(summary, gifs, generationOptions{outputFormat: "gif"})
	}
	writeResult(stdout, summary)
	return nil
}
