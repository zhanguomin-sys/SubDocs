package connect

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// installTimeout bounds one run of the image tool's installer. It only copies
// files it carries, so it normally finishes in a second or two.
const installTimeout = 3 * time.Minute

// BundledImageTool is a released image tool program carried inside the
// assistant. Install writes it to CacheDir once (checked against SHA256) and
// runs "<program> install" with the key on standard input, as a customer
// would type it in a terminal.
type BundledImageTool struct {
	Program  []byte
	Release  string // e.g. 0.8.2
	SHA256   string // hex of Program, from the release's SHA256SUMS
	CacheDir string
}

func (b *BundledImageTool) Version() string {
	if b == nil || len(b.Program) == 0 {
		return ""
	}
	return b.Release
}

// Verify reports whether Program matches the release checksum.
func (b *BundledImageTool) Verify() error {
	sum := sha256.Sum256(b.Program)
	if !strings.EqualFold(hex.EncodeToString(sum[:]), strings.TrimSpace(b.SHA256)) {
		return errors.New("内置的生图程序校验不通过，请重新下载助手")
	}
	return nil
}

func (b *BundledImageTool) programPath() string {
	name := "myautoapi-imagines-tool-v" + b.Release
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(b.CacheDir, name)
}

// unpack writes the program unless an identical copy is already there.
func (b *BundledImageTool) unpack() (string, error) {
	if err := b.Verify(); err != nil {
		return "", err
	}
	path := b.programPath()
	if existing, err := os.ReadFile(path); err == nil && bytes.Equal(existing, b.Program) {
		return path, nil
	}
	if err := os.MkdirAll(b.CacheDir, 0o700); err != nil {
		return "", err
	}
	temporary, err := os.CreateTemp(b.CacheDir, ".image-tool-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(temporary.Name())
	if _, err := temporary.Write(b.Program); err != nil {
		temporary.Close()
		return "", err
	}
	if err := temporary.Close(); err != nil {
		return "", err
	}
	if err := os.Chmod(temporary.Name(), 0o700); err != nil {
		return "", err
	}
	if err := os.Rename(temporary.Name(), path); err != nil {
		// A copy still running from an earlier install keeps the old file
		// locked on Windows; an identical one is fine to use.
		if existing, readErr := os.ReadFile(path); readErr == nil && bytes.Equal(existing, b.Program) {
			return path, nil
		}
		return "", err
	}
	return path, nil
}

// Install runs the image tool's installer for codexHome with key.
func (b *BundledImageTool) Install(ctx context.Context, codexHome, key string) error {
	if b.Version() == "" {
		return errors.New("助手没有内置生图工具")
	}
	program, err := b.unpack()
	if err != nil {
		return errors.New("无法准备生图程序：" + reason(err))
	}
	ctx, cancel := context.WithTimeout(ctx, installTimeout)
	defer cancel()
	command := exec.CommandContext(ctx, program, "install")
	command.Env = append(withoutCodexHome(os.Environ()), "CODEX_HOME="+codexHome)
	command.Stdin = strings.NewReader(key + "\n")
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	hideConsole(command)
	if err := command.Run(); err != nil {
		if ctx.Err() != nil {
			return errors.New("生图安装程序没有在 3 分钟内完成")
		}
		if message := installFailure(stderr.String(), key); message != "" {
			return errors.New(message)
		}
		return errors.New("生图安装程序出错：" + reason(err))
	}
	return nil
}

// installFailure picks the "安装失败：…" line the image tool prints, and makes
// sure the key never reaches the window even if a message quoted it.
func installFailure(output, key string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "安装失败："); ok && rest != "" {
			if key != "" {
				rest = strings.ReplaceAll(rest, key, MaskKey(key))
			}
			return rest
		}
	}
	return ""
}

func withoutCodexHome(env []string) []string {
	kept := env[:0:0]
	for _, entry := range env {
		if !strings.HasPrefix(strings.ToUpper(entry), "CODEX_HOME=") {
			kept = append(kept, entry)
		}
	}
	return kept
}
