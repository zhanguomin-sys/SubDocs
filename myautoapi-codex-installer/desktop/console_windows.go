package main

import "syscall"

func configureConsole() {
	kernel := syscall.NewLazyDLL("kernel32.dll")
	_, _, _ = kernel.NewProc("SetConsoleOutputCP").Call(65001)
	_, _, _ = kernel.NewProc("SetConsoleCP").Call(65001)
}
