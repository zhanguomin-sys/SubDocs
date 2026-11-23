//go:build !quickconnect_bundle

package main

import "io/fs"

// Repository-only tests can run before generating the Windows release assets.
var embeddedPackages fs.FS
