package main

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"

	"myautoapi-quick-connect/connect"
)

func TestVersionInfoNamesTheProduct(t *testing.T) {
	info := VersionInfo()
	if languages := info.Table(); len(languages) != 1 || languages[LangChinese] == nil {
		t.Fatalf("want exactly one zh-CN translation, got %v", languages)
	}
	table := info.Table().GetMainTranslation()
	for key, want := range map[string]string{
		version.ProductName:     "MyAutoAPI中转快速接入助手",
		version.FileDescription: "MyAutoAPI中转快速接入助手",
		version.ProductVersion:  connect.Version,
		version.FileVersion:     connect.Version,
	} {
		if table[key] != want {
			t.Errorf("%s = %q, want %q", key, table[key], want)
		}
	}
}

// TestBuiltExecutable checks the exe produced by ../desktop/build.py:
// x64 GUI program, under the 25 MiB site limit, icon and file properties.
// Set QUICK_CONNECT_EXE to check another file.
func TestBuiltExecutable(t *testing.T) {
	path := os.Getenv("QUICK_CONNECT_EXE")
	if path == "" {
		path = filepath.Join("..", "..", "..", "dist", "desktop", "Windows-x64", "1.MyAutoAPI中转快速接入助手.exe")
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		t.Skip("还没有构建 exe：先运行 ../desktop/build.py")
	}
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > 25*1024*1024 {
		t.Fatalf("exe is %d bytes, over 25 MiB", len(data))
	}
	if !bytes.HasPrefix(data, []byte("MZ")) {
		t.Fatal("not a PE file")
	}
	pe := binary.LittleEndian.Uint32(data[0x3c:])
	if string(data[pe:pe+4]) != "PE\x00\x00" || binary.LittleEndian.Uint16(data[pe+4:]) != 0x8664 {
		t.Fatal("not an x64 PE file")
	}
	optional := pe + 24
	if binary.LittleEndian.Uint16(data[optional:]) != 0x20b || binary.LittleEndian.Uint16(data[optional+68:]) != 2 {
		t.Fatal("not a PE32+ Windows GUI program")
	}

	resources, err := winres.LoadFromEXE(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	translations := 0
	resources.WalkType(winres.RT_VERSION, func(winres.Identifier, uint16, []byte) bool {
		translations++
		return true
	})
	raw := resources.Get(winres.RT_VERSION, winres.ID(1), LangChinese)
	if raw == nil || translations != 1 {
		t.Fatalf("want one zh-CN version resource, found %d", translations)
	}
	info, err := version.FromBytes(raw)
	if err != nil {
		t.Fatal(err)
	}
	table := info.Table().GetMainTranslation()
	if table[version.ProductName] != connect.ProductName || table[version.FileDescription] != connect.ProductName ||
		table[version.ProductVersion] != connect.Version || table[version.FileVersion] != connect.Version {
		t.Fatalf("version strings %v", table)
	}
	icon, err := resources.GetIcon(winres.ID(iconID))
	if err != nil {
		t.Fatal(err)
	}
	var icoFile bytes.Buffer
	if err := icon.SaveICO(&icoFile); err != nil {
		t.Fatal(err)
	}
	ico := icoFile.Bytes()
	count := int(binary.LittleEndian.Uint16(ico[4:]))
	sizes := map[int]bool{}
	for i := 0; i < count; i++ {
		width := int(ico[6+16*i])
		if width == 0 {
			width = 256
		}
		sizes[width] = true
	}
	for _, size := range []int{16, 32, 48, 256} {
		if !sizes[size] {
			t.Errorf("icon lacks %dx%d (has %v)", size, size, sizes)
		}
	}
	manifest := resources.Get(winres.RT_MANIFEST, winres.ID(1), winres.LCIDDefault)
	if !bytes.Contains(manifest, []byte(`level="asInvoker"`)) {
		t.Fatalf("manifest must not ask for administrator rights:\n%s", manifest)
	}
}
