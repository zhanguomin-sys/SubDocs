package connect

import (
	"os/exec"
	"syscall"
)

// hideConsole keeps the image tool, a console program, from flashing a
// terminal window over the assistant.
func hideConsole(command *exec.Cmd) {
	const createNoWindow = 0x08000000
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
}
