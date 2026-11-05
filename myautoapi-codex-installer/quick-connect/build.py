#!/usr/bin/env python3
"""Build the quick-connect assistant's native programs.

build_windows(): the Windows window (cross-compiled anywhere).
build_mac(): the launcher of the Mac terminal .app (macapp/), which receives
myautoapi:// links through Cocoa and so must be built on a Mac; the
mac-quick-connect GitHub Actions workflow does that.
../desktop/build.py places both into the numbered tool set.
"""

from __future__ import annotations

import os
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent
WORK = ROOT / ".work"
BUILD_TAGS = "desktop,production,wv2runtime.browser"
MAC_RUNNER = ROOT / "macapp" / "run.command"
MAX_DOWNLOAD_BYTES = 25 * 1024 * 1024
# The image tool release the window carries. Its source lives in
# myautoapi-imagines-tool/; only the published program is copied in here.
IMAGE_TOOL_RELEASE = "0.8.3"
IMAGE_TOOL_RELEASES = ROOT.parent.parent / "docs" / "public" / "install" / "myautoapi-imagines-tool" / "releases"
IMAGE_TOOL_BUNDLE = ROOT / "imagetool"
MODELS_SCRIPT = ROOT.parent / "install" / "codex-models.ps1"
MODELS_PUBLISHED = ROOT.parent.parent / "docs" / "public" / "install" / "codex-models.ps1"
MODELS_BUNDLE = ROOT / "modelstool"


def go_command() -> str:
    command = shutil.which("go") or "/usr/local/go/bin/go"
    if not Path(command).is_file():
        raise SystemExit("构建需要本机已有 Go 1.25 或以上版本。")
    return command


def environment(**extra: str) -> dict[str, str]:
    """Keep module downloads, caches and temporary files inside .work/."""
    env = os.environ.copy()
    for name, directory in {
        "GOPATH": WORK / "go-path",
        "GOCACHE": WORK / "go-cache",
        "GOTMPDIR": WORK / "tmp",
        "TMPDIR": WORK / "tmp",
        "XDG_CACHE_HOME": WORK / "cache",
        "XDG_CONFIG_HOME": WORK / "config",
    }.items():
        directory.mkdir(parents=True, exist_ok=True)
        env[name] = str(directory)
    env.update({"GOTOOLCHAIN": "local", "GOFLAGS": "-mod=readonly -modcacherw", "CGO_ENABLED": "0", "GOENV": "off"})
    env.update(extra)
    return env


def bundle_image_tool(platform: str) -> None:
    """Copy the published image tool program for platform into imagetool/.

    The checksum comes from the release's SHA256SUMS and is checked twice:
    here, and by the window before it runs the program.
    """
    import hashlib

    release = IMAGE_TOOL_RELEASES / f"v{IMAGE_TOOL_RELEASE}"
    name = f"myautoapi-imagines-tool-v{IMAGE_TOOL_RELEASE}-{platform}"
    sums = {}
    for line in (release / "SHA256SUMS").read_text(encoding="utf-8").splitlines():
        parts = line.split()
        if len(parts) == 2:
            sums[parts[1].lstrip("*")] = parts[0].lower()
    if name not in sums:
        raise SystemExit(f"{release / 'SHA256SUMS'} 里没有 {name}。")
    program = (release / name).read_bytes()
    if hashlib.sha256(program).hexdigest() != sums[name]:
        raise SystemExit(f"{name} 与 SHA256SUMS 不一致，停止打包。")
    IMAGE_TOOL_BUNDLE.mkdir(exist_ok=True)
    (IMAGE_TOOL_BUNDLE / "program").write_bytes(program)
    (IMAGE_TOOL_BUNDLE / "VERSION").write_text(IMAGE_TOOL_RELEASE + "\n", encoding="utf-8")
    (IMAGE_TOOL_BUNDLE / "SHA256").write_text(sums[name] + "\n", encoding="utf-8")


def bundle_models_script() -> None:
    """Copy install/codex-models.ps1 into modelstool/ after checking it is the
    published script, so the window runs what the site serves. The published
    copy has no UTF-8 BOM (for irm | iex); the bundled one keeps it, which
    Windows PowerShell 5.1 needs to read a Chinese script from a file."""
    bom = b"\xef\xbb\xbf"
    script = MODELS_SCRIPT.read_bytes()
    if not script.startswith(bom):
        raise SystemExit(f"{MODELS_SCRIPT} 缺少 UTF-8 BOM，Windows PowerShell 5.1 读中文会乱码。")
    if script.removeprefix(bom) != MODELS_PUBLISHED.read_bytes().removeprefix(bom):
        raise SystemExit(f"{MODELS_SCRIPT} 与发布目录的 {MODELS_PUBLISHED.name} 不一致，先按国产模型子项目的流程同步。")
    MODELS_BUNDLE.mkdir(exist_ok=True)
    (MODELS_BUNDLE / "codex-models.ps1").write_bytes(script)


def build_windows(output: Path) -> Path:
    """Write the x64 exe with icon, manifest, version information and the
    bundled image tool."""
    go = go_command()
    bundle_image_tool("windows-amd64.exe")
    bundle_models_script()
    subprocess.run([go, "run", "./tools/resources", "-icon", "frontend/logo.png", "-out", "rsrc_windows_amd64.syso"],
                   cwd=ROOT, env=environment(), check=True)
    output.parent.mkdir(parents=True, exist_ok=True)
    subprocess.run([
        go, "build", "-trimpath", "-buildvcs=false", "-tags", BUILD_TAGS,
        "-ldflags", "-s -w -H windowsgui", "-o", str(output), ".",
    ], cwd=ROOT, env=environment(GOOS="windows", GOARCH="amd64"), check=True)
    size = output.stat().st_size
    if size > MAX_DOWNLOAD_BYTES:
        raise SystemExit(f"{output.name} 有 {size} 字节，超过文档站 25 MiB 的单文件上限。")
    output.chmod(0o755)
    return output


def build_mac(output: Path) -> Path:
    """Write the arm64 launcher; needs macOS with the Xcode command line tools."""
    if sys.platform != "darwin":
        raise SystemExit("Mac 版启动器用到 Apple 的系统接口，只能在 Mac 上编译；请手动触发 GitHub Actions 的 mac-quick-connect 打包。")
    output.parent.mkdir(parents=True, exist_ok=True)
    subprocess.run([
        go_command(), "build", "-trimpath", "-buildvcs=false", "-ldflags", "-s -w", "-o", str(output), "./macapp",
    ], cwd=ROOT, env=environment(GOOS="darwin", GOARCH="arm64", CGO_ENABLED="1", MACOSX_DEPLOYMENT_TARGET="12.0"), check=True)
    output.chmod(0o755)
    return output


if __name__ == "__main__":
    target = Path(sys.argv[1]) if len(sys.argv) > 1 else WORK / "myautoapi-quick-connect-windows-x64.exe"
    print(build_windows(target.resolve()))
