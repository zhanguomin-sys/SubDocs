package connect

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestReadModelsStatus(t *testing.T) {
	inUse := "model_catalog_json = \"myautoapi-models.json\"\n"
	cases := []struct {
		name, managed, config string
		want                  ModelsStatus
	}{
		{"never imported", "", "", ModelsStatus{State: ModelsNone}},
		{"imported, relative path", `["deepseek-flash","glm-5.3","kimi-k3"]`, inUse, ModelsStatus{State: ModelsActive, Count: 3}},
		{"absolute path", `["glm-5.3"]`, "model_catalog_json = 'HOME/myautoapi-models.json'\n", ModelsStatus{State: ModelsActive, Count: 1}},
		{"gpt ids are not counted", `["gpt-6.1-terra","glm-5.3"]`, inUse, ModelsStatus{State: ModelsActive, Count: 1}},
		{"Codex reconnected, catalog dropped", `["glm-5.3","kimi-k3"]`, "model = \"gpt-6.1-sol\"\n", ModelsStatus{State: ModelsInactive, Count: 2}},
		{"points elsewhere", `["glm-5.3"]`, "model_catalog_json = \"other.json\"\n", ModelsStatus{State: ModelsInactive, Count: 1}},
		{"removed", `[]`, inUse, ModelsStatus{State: ModelsNone}},
		{"damaged record", `{`, inUse, ModelsStatus{State: ModelsNone}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			home := t.TempDir()
			if c.managed != "" {
				writeFile(t, filepath.Join(home, modelsManagedName), "\ufeff"+c.managed+"\n")
			}
			if c.config != "" {
				config := strings.ReplaceAll(c.config, "HOME", filepath.ToSlash(home))
				writeFile(t, filepath.Join(home, "config.toml"), config)
			}
			if got := ReadModelsStatus(home); got != c.want {
				t.Fatalf("got %+v want %+v", got, c.want)
			}
		})
	}
}

// fakePowerShell stands in for powershell.exe: it records the environment
// and prints what the script would.
func fakePowerShell(t *testing.T, body string) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("用 sh 模拟 PowerShell")
	}
	path := filepath.Join(t.TempDir(), "powershell")
	writeFile(t, path, "#!/bin/sh\n"+body+"\n")
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}

func modelsService(t *testing.T, shell string) (*Service, string) {
	home := filepath.Join(t.TempDir(), ".codex")
	return &Service{
		CodexHome: home, ModelsHome: home, Now: time.Now,
		Models: &BundledModelsScript{Script: []byte("# script"), CacheDir: t.TempDir(), PowerShell: shell},
	}, home
}

func TestImportModelsResults(t *testing.T) {
	success := fakePowerShell(t, `[ "$MYAUTOAPI_MODELS_UNINSTALL" = 0 ] || exit 9
[ -f "$MYAUTOAPI_MODELS_SCRIPT" ] || exit 8
printf '\357\273\277成功导入后会自动关闭 ChatGPT/Codex，运行中的任务会中断。\r\n已导入 11 个国产模型，原 GPT 和其他模型已保留。\r\n已关闭 Codex。\r\n\r\n重新打开 App，新建任务后选择模型；API Key 分组由你在中转站切换。\r\n'`)
	service, _ := modelsService(t, success)
	got := service.ImportModels(context.Background(), false)
	want := "已导入 11 个国产模型，原 GPT 和其他模型已保留。\n已关闭 Codex。\n重新打开 App，新建任务后选择模型；API Key 分组由你在中转站切换。"
	if got.Tone != ToneSuccess || got.Message != want {
		t.Fatalf("got %s %q", got.Tone, got.Message)
	}

	failure := fakePowerShell(t, `echo '成功导入后会自动关闭 ChatGPT/Codex，运行中的任务会中断。'
echo '读取 Codex 配置失败：拒绝访问。'
echo 'At line:1 char:1' >&2
exit 1`)
	service, _ = modelsService(t, failure)
	got = service.ImportModels(context.Background(), false)
	if got.Tone != ToneError || got.Message != "读取 Codex 配置失败：拒绝访问。\n"+MessageModelsFailed {
		t.Fatalf("failure: %s %q", got.Tone, got.Message)
	}

	remove := fakePowerShell(t, `[ "$MYAUTOAPI_MODELS_UNINSTALL" = 1 ] || exit 9
echo '移除当前启用目录中的国产模型后会自动关闭 ChatGPT/Codex，运行中的任务会中断。'
echo '已移除本工具导入的国产模型，原 GPT 和其他模型已保留。'`)
	service, _ = modelsService(t, remove)
	got = service.ImportModels(context.Background(), true)
	if got.Tone != ToneSuccess || got.Message != "已移除本工具导入的国产模型，原 GPT 和其他模型已保留。" {
		t.Fatalf("remove: %s %q", got.Tone, got.Message)
	}

	empty := &Service{ModelsHome: t.TempDir(), Now: time.Now}
	if got := empty.ImportModels(context.Background(), false); got.Message != MessageModelsNotBundled {
		t.Fatalf("not bundled: %q", got.Message)
	}
}

func TestOutputLinesReadsGBK(t *testing.T) {
	encoded, err := simplifiedchinese.GBK.NewEncoder().String("已导入 11 个国产模型\r\n")
	if err != nil {
		t.Fatal(err)
	}
	if got := outputLines([]byte(encoded)); len(got) != 1 || got[0] != "已导入 11 个国产模型" {
		t.Fatalf("got %q", got)
	}
}

// TestPowerShellWrapperPassesArguments runs the real wrapper in PowerShell 7
// against a stand-in script, with a Chinese path, to check the parameters
// and the switch arrive as codex-models.ps1 expects them.
func TestPowerShellWrapperPassesArguments(t *testing.T) {
	shell := powerShell(t)
	stand := "\ufeffparam([string]$CodexHomePath, [string]$CodexPath, [switch]$Uninstall)\nWrite-Host (\"家目录=\" + $CodexHomePath)\nWrite-Host (\"移除=\" + $Uninstall.IsPresent)\n"
	for _, uninstall := range []bool{false, true} {
		home := filepath.Join(t.TempDir(), "用户 O'Brien", ".codex")
		script := &BundledModelsScript{Script: []byte(stand), CacheDir: t.TempDir(), PowerShell: shell}
		lines, err := script.Run(context.Background(), home, uninstall)
		if err != nil {
			t.Fatalf("%v %q", err, lines)
		}
		want := []string{"家目录=" + home, "移除=False"}
		if uninstall {
			want[1] = "移除=True"
		}
		if strings.Join(lines, "\n") != strings.Join(want, "\n") {
			t.Fatalf("got %q want %q", lines, want)
		}
	}
}
