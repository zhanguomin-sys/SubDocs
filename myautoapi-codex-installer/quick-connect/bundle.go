//go:build quickconnect_bundle

package main

import (
	"embed"
	"io/fs"
)

// Files are copied from the MyAutoAPI release tree by build.py.
//
//go:embed packages/release.json packages/image-tool.exe packages/models.ps1 packages/official-models.json
var toolPackageFiles embed.FS

var embeddedPackages fs.FS = toolPackageFiles
