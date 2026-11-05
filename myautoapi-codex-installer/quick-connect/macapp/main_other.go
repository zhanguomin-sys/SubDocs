//go:build !darwin

package main

import (
	"fmt"
	"os"
)

// The launcher only runs inside the Mac .app; elsewhere only its tests run.
func main() {
	fmt.Fprintln(os.Stderr, "此启动器只用于 Mac 版 MyAutoAPI中转快速接入助手。")
	os.Exit(1)
}
