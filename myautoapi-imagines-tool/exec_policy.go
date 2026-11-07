package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const execPolicyLogicalPath = "rules/myautoapi-imagines-tool.rules"

func installedToolPath(layout installLayout) string {
	path, err := filepath.Abs(layout.binaryPath)
	if err == nil {
		return path
	}
	return filepath.Clean(layout.binaryPath)
}

func bashPathExpression(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'"
}

func powerShellPathExpression(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "''") + "'"
}

func renderInstalledSkill(layout installLayout) string {
	return renderSkillTemplate(embeddedSkill, layout)
}

func renderSkillTemplate(template string, layout installLayout) string {
	path := installedToolPath(layout)
	replacer := strings.NewReplacer(
		"{{MYAUTOAPI_IMAGE_TOOL_BASH}}", bashPathExpression(path),
		"{{MYAUTOAPI_IMAGE_TOOL_POWERSHELL}}", powerShellPathExpression(path),
		"{{MYAUTOAPI_IMAGE_TOOL}}", path,
	)
	return replacer.Replace(template)
}

func renderExecPolicyRules(layout installLayout) string {
	// Scope each rule to one supported operation on the installed executable.
	// Installation and uninstallation remain outside the automatic allow list.
	var result strings.Builder
	result.WriteString("# Managed by myautoapi-imagines-tool; loaded when Codex starts.\n")
	for _, command := range []string{"generate", "edit", "models", "paths", "doctor", "recover", "version"} {
		result.WriteString("prefix_rule(\n" +
			"    pattern = [" + strconv.Quote(installedToolPath(layout)) + ", " + strconv.Quote(command) + "],\n" +
			"    decision = \"allow\",\n" +
			"    justification = \"运行已安装的 MyAutoAPI 生图工具\",\n" +
			")\n")
	}
	return result.String()
}

func checkExecPolicyTarget(layout installLayout, state *installState) error {
	if err := ensureDirectoryTarget(layout.rulesDir, "Codex 放行规则目录"); err != nil {
		return err
	}
	info, err := os.Lstat(layout.rulesPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("Codex 放行规则不是可管理的普通文件：%s", layout.rulesPath)
	}
	if state == nil || state.Files[execPolicyLogicalPath] == "" {
		return fmt.Errorf("Codex 放行规则无法确认由本工具管理，停止覆盖：%s", layout.rulesPath)
	}
	return nil
}
