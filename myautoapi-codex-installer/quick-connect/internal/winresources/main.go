// Command winresources packages the application's own icon into a Go linker object.
package main

import (
	"errors"
	"log"
	"os"

	"github.com/tc-hib/winres"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) != 3 {
		log.Fatal("usage: winresources <app.ico> <windows-amd64.syso>")
	}
	if err := writeObject(os.Args[1], os.Args[2]); err != nil {
		log.Fatal(err)
	}
}

func writeObject(iconPath, objectPath string) error {
	src, err := os.Open(iconPath)
	if err != nil {
		return err
	}
	defer src.Close()
	icon, err := winres.LoadICO(src)
	if err != nil {
		return err
	}

	var resources winres.ResourceSet
	// Wails loads resource 3 for the title bar. Its window class loads
	// IDI_APPLICATION (32512) for the large icon used by Windows.
	for _, id := range []winres.ID{3, 32512} {
		if err := resources.SetIcon(id, icon); err != nil {
			return err
		}
	}
	out, err := os.Create(objectPath)
	if err != nil {
		return err
	}
	writeErr := resources.WriteObject(out, winres.ArchAMD64)
	return errors.Join(writeErr, out.Close())
}
