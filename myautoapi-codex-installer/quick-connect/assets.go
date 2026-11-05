package main

import (
	"embed"
	"io/fs"
)

//go:embed all:frontend
var embeddedFrontend embed.FS

// frontendFiles is the window's page: plain HTML, CSS and JavaScript with no
// build step.
func frontendFiles() fs.FS {
	files, err := fs.Sub(embeddedFrontend, "frontend")
	if err != nil {
		panic(err)
	}
	return files
}
