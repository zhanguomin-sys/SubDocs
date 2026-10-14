package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

const generationFailureMessage = "生成失败。请到后台查看是否有生成记录，或让 AI 修复生图工具。"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) == 0 {
		writeResult(stderr, map[string]any{"ok": false, "error": "缺少命令：install、uninstall、generate、edit 或 version。"})
		return 2
	}

	var err error
	switch arguments[0] {
	case "install":
		options := installOptions{}
		for _, argument := range arguments[1:] {
			switch argument {
			case "--integrated":
				options.integrated = true
				options.compareProgrammingKey = true
			default:
				err = fmt.Errorf("安装参数无效：%s", argument)
			}
		}
		if err == nil {
			err = installWithOptions(stdout, os.Stdin, options)
		}
	case "uninstall":
		err = uninstall(stdout)
	case "generate":
		err = generate(arguments[1:], stdout)
	case "edit":
		err = edit(arguments[1:], stdout)
	case "version":
		_, err = fmt.Fprintln(stdout, version())
	default:
		err = fmt.Errorf("未知命令：%s", arguments[0])
	}
	if err == nil {
		return 0
	}

	if arguments[0] == "install" || arguments[0] == "uninstall" {
		action := "安装"
		if arguments[0] == "uninstall" {
			action = "卸载"
		}
		fmt.Fprintf(stderr, "%s失败：%s\n", action, redactError(err.Error(), ""))
		fmt.Fprintln(stderr, "本次操作未完成；写入前已停止或已恢复本次修改。")
		return 1
	}

	diagnostic := redactError(err.Error(), "")
	if arguments[0] == "generate" || arguments[0] == "edit" {
		writeResult(stderr, map[string]any{
			"ok":         false,
			"error":      generationFailureMessage,
			"diagnostic": diagnostic,
		})
		return 1
	}
	writeResult(stderr, map[string]any{"ok": false, "error": diagnostic})
	return 1
}

func writeResult(destination io.Writer, value any) {
	encoder := json.NewEncoder(destination)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(value)
}
