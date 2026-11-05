// Command resources writes the Windows resources of the assistant into a
// .syso file that go build links in: icon, manifest and the version
// information shown in the file properties and by the browser.
package main

import (
	"flag"
	"fmt"
	"image/png"
	"os"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"

	"myautoapi-quick-connect/connect"
)

// Wails loads the window icon from resource ID 3; Explorer shows the same
// group because it is the only one.
const iconID = 3

// LangChinese is zh-CN, the language of the version strings.
const LangChinese = 0x0804

// FileName is the name the build gives the exe for download.
const FileName = "myautoapi-quick-connect-windows-x64.exe"

func main() {
	iconPath := flag.String("icon", "frontend/logo.png", "PNG used for the icon")
	output := flag.String("out", "rsrc_windows_amd64.syso", "object file to write")
	flag.Parse()
	if err := write(*iconPath, *output); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func write(iconPath, output string) error {
	file, err := os.Open(iconPath)
	if err != nil {
		return err
	}
	image, err := png.Decode(file)
	file.Close()
	if err != nil {
		return err
	}
	icon, err := winres.NewIconFromResizedImage(image, []int{256, 128, 64, 48, 32, 16})
	if err != nil {
		return err
	}
	resources := winres.ResourceSet{}
	if err := resources.SetIcon(winres.ID(iconID), icon); err != nil {
		return err
	}
	resources.SetManifest(winres.AppManifest{
		ExecutionLevel:      winres.AsInvoker,
		DPIAwareness:        winres.DPIPerMonitorV2,
		UseCommonControlsV6: true,
		LongPathAware:       true,
	})
	resources.SetVersionInfo(VersionInfo())

	out, err := os.Create(output)
	if err != nil {
		return err
	}
	writeErr := resources.WriteObject(out, winres.ArchAMD64)
	closeErr := out.Close()
	if writeErr != nil {
		return writeErr
	}
	return closeErr
}

// VersionInfo names the product in the file properties. Browsers show
// FileDescription in their "open this application?" prompt.
func VersionInfo() version.Info {
	var info version.Info
	for key, value := range map[string]string{
		version.ProductName:      connect.ProductName,
		version.FileDescription:  connect.ProductName,
		version.CompanyName:      "MyAutoAPI",
		version.LegalCopyright:   "© MyAutoAPI",
		version.FileVersion:      connect.Version,
		version.ProductVersion:   connect.Version,
		version.InternalName:     "myautoapi-quick-connect",
		version.OriginalFilename: FileName,
	} {
		if err := info.Set(LangChinese, key, value); err != nil {
			panic(err)
		}
	}
	// After the strings, so the numeric versions go into the one zh-CN
	// translation instead of creating a neutral one Explorer would pick.
	info.SetFileVersion(connect.Version)
	info.SetProductVersion(connect.Version)
	return info
}
