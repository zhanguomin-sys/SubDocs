//go:build !windows

package connect

import "os/exec"

func hideConsole(*exec.Cmd) {}
