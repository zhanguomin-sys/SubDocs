package main

import (
	"embed"
	"strings"

	"myautoapi-quick-connect/connect"
)

// The image tool program build.py places in imagetool/ (see its README).
// Without it the window still works and says the image tool is not bundled.
//
//go:embed all:imagetool
var imageBundle embed.FS

func bundledImageTool(cacheDir string) *connect.BundledImageTool {
	program, err := imageBundle.ReadFile("imagetool/program")
	if err != nil {
		return nil
	}
	release, _ := imageBundle.ReadFile("imagetool/VERSION")
	sum, _ := imageBundle.ReadFile("imagetool/SHA256")
	return &connect.BundledImageTool{
		Program:  program,
		Release:  strings.TrimSpace(string(release)),
		SHA256:   strings.TrimSpace(string(sum)),
		CacheDir: cacheDir,
	}
}
