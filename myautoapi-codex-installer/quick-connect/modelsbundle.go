package main

import (
	"embed"

	"myautoapi-quick-connect/connect"
)

// The domestic model script build.py places in modelstool/ (see its README).
//
//go:embed all:modelstool
var modelsBundle embed.FS

func bundledModelsScript(cacheDir string) *connect.BundledModelsScript {
	script, err := modelsBundle.ReadFile("modelstool/codex-models.ps1")
	if err != nil {
		return nil
	}
	return &connect.BundledModelsScript{Script: script, CacheDir: cacheDir}
}
