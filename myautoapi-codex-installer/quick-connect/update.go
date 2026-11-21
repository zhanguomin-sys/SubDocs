package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const manifestPath = "/download/myautoapi-quick-connect-windows-x64.json"

var (
	versionPattern = regexp.MustCompile("^[0-9]{1,4}\\.[0-9]{1,4}\\.[0-9]{1,4}$")
	hashPattern    = regexp.MustCompile("^[0-9a-f]{64}$")
)

type Release struct {
	Version string `json:"version"`
	Path    string `json:"path"`
	SHA256  string `json:"sha256"`
	Size    int64  `json:"size"`
}

func (r Release) filename() string {
	return "myautoapi-quick-connect-" + r.Version + "-windows-x64.exe"
}

func (r Release) valid() bool {
	return versionPattern.MatchString(r.Version) &&
		r.Path == "/download/quick-connect/"+r.filename() &&
		hashPattern.MatchString(r.SHA256) &&
		r.Size >= 1024 && r.Size <= 24<<20
}

func (r Release) newer() bool {
	if !r.valid() {
		return false
	}
	have, latest := strings.Split(version, "."), strings.Split(r.Version, ".")
	for i := range have {
		x, _ := strconv.Atoi(have[i])
		y, _ := strconv.Atoi(latest[i])
		if x != y {
			return y > x
		}
	}
	return false
}

type updater struct {
	http   *http.Client
	origin string
}

func trustedClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func (u updater) readLatest(ctx context.Context) (Release, error) {
	var latest Release
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.origin+manifestPath, nil)
	if err != nil {
		return latest, err
	}
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := u.http.Do(req)
	if err != nil {
		return latest, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return latest, fmt.Errorf("更新清单返回 HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4097))
	if err != nil || len(data) > 4096 || json.Unmarshal(data, &latest) != nil || !latest.valid() {
		return Release{}, errors.New("更新清单无效")
	}
	return latest, nil
}

func (u updater) download(ctx context.Context, latest Release, path string) error {
	if !latest.newer() {
		return errors.New("没有可用的新版本")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.origin+latest.Path, nil)
	if err != nil {
		return err
	}
	resp, err := u.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || (resp.ContentLength >= 0 && resp.ContentLength != latest.Size) {
		return errors.New("下载地址或文件大小不符")
	}

	f, err := os.CreateTemp(filepath.Dir(path), ".myautoapi-download-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	sum := sha256.New()
	signature := make([]byte, 2)
	_, err = io.ReadFull(resp.Body, signature)
	var size int64
	if err == nil {
		writer := io.MultiWriter(f, sum)
		if _, err = writer.Write(signature); err == nil {
			size, err = io.Copy(writer, io.LimitReader(resp.Body, latest.Size-1))
		}
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if size+2 != latest.Size || string(signature) != "MZ" || hex.EncodeToString(sum.Sum(nil)) != latest.SHA256 {
		return errors.New("下载文件校验失败，请重新检查更新")
	}
	return os.Rename(f.Name(), path)
}
