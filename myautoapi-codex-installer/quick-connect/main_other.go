//go:build !windows

package main

import (
	"fmt"
	"os"

	"myautoapi-quick-connect/connect"
)

// The window is Windows only; Mac users get the terminal .app built from
// ../desktop, which runs install/codex.sh.
func main() {
	fmt.Fprintln(os.Stderr, connect.ProductName+" 的窗口版只支持 Windows。")
	os.Exit(1)
}
