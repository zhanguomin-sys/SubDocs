#!/usr/bin/env python3
"""Build the numbered tools and website downloads.

The current Mac quick-connect app is a cross-platform-buildable terminal app;
--terminal-mac enables this mode. The optional Cocoa protocol launcher has
separate --mac-only and --mac-launcher build modes.
"""

import argparse
import importlib.util
import shutil
import sys
from pathlib import Path

from packaging import (WORK_ROOT, mac_application, powershell_runner, windows_executable,
                       write_checksums, zip_application)

PROJECT_ROOT = Path(__file__).resolve().parents[1]
OUTPUT_ROOT = PROJECT_ROOT / "dist" / "desktop"
WINDOWS = OUTPUT_ROOT / "Windows-x64"
MAC = OUTPUT_ROOT / "Mac-Apple芯片"
DOWNLOAD = OUTPUT_ROOT / "download"

ASSISTANT = "MyAutoAPI中转快速接入助手"
ASSISTANT_TOOL = "1." + ASSISTANT
# Same identifier as the former "1.Codex接入配置", so the permission to
# control Terminal carries over.
ASSISTANT_IDENTIFIER = "com.myautoapi.codex-config"
ASSISTANT_VERSION = "1.1.4"
WINDOWS_DOWNLOAD = "myautoapi-quick-connect-windows-x64.exe"
MAC_DOWNLOAD = "myautoapi-quick-connect-mac-arm64.zip"

TERMINAL_TOOLS = (
    ("3.Codex国产模型导入", "codex-models", "com.myautoapi.codex-models"),
)


def load_quick_connect():
    spec = importlib.util.spec_from_file_location("quick_connect_build", PROJECT_ROOT / "quick-connect" / "build.py")
    module = importlib.util.module_from_spec(spec)
    previous = sys.dont_write_bytecode
    sys.dont_write_bytecode = True
    try:
        spec.loader.exec_module(module)
    finally:
        sys.dont_write_bytecode = previous
    return module


# Registers myautoapi:// links with Launch Services when the .app is first seen.
URL_TYPES = {"CFBundleURLTypes": [{
    "CFBundleURLName": "com.myautoapi.import",
    "CFBundleURLSchemes": ["myautoapi"],
}]}


def assistant_mac(output: Path, launcher: Path | None, runner: str) -> Path:
    return mac_application(output, ASSISTANT_IDENTIFIER, ASSISTANT_VERSION, {
        "install.sh": (PROJECT_ROOT / "install" / "codex.sh").read_bytes(),
    }, runner=runner, launcher=launcher, extra_info=(URL_TYPES if launcher else {"MyAutoAPILauncherMode": "terminal"}))


def check_mac_launcher(path: Path) -> Path:
    header = path.read_bytes()[:8]
    if header != bytes.fromhex("cffaedfe0c000001"):
        raise SystemExit(f"{path} 不是 Apple 芯片 Mac 程序。")
    return path


def main() -> None:
    parser = argparse.ArgumentParser()
    group = parser.add_mutually_exclusive_group()
    group.add_argument("--mac-only", action="store_true", help="在 Mac 上只构建 Mac 版快速接入助手")
    group.add_argument("--mac-launcher", type=Path, help="Actions 构建好的 Mac 启动器（.app 内 Contents/MacOS/launcher）")
    group.add_argument("--terminal-mac", action="store_true", help="跨平台构建 Mac 剪贴板接入版，不注册网页导入协议")
    options = parser.parse_args()

    quick_connect = load_quick_connect()
    if f'Version     = "{ASSISTANT_VERSION}"' not in (PROJECT_ROOT / "quick-connect/connect/product.go").read_text():
        raise SystemExit("Mac 版本号与 quick-connect/connect/product.go 不一致。")
    if options.terminal_mac:
        mac_launcher = None
    elif options.mac_launcher:
        # Copied aside first: the given file may sit inside OUTPUT_ROOT.
        mac_launcher = WORK_ROOT / "mac-launcher"
        mac_launcher.parent.mkdir(parents=True, exist_ok=True)
        shutil.copyfile(check_mac_launcher(options.mac_launcher.resolve()), mac_launcher)
    elif sys.platform == "darwin":
        mac_launcher = None
    else:
        raise SystemExit("Mac 版快速接入助手只能在 Mac 上编译：请手动触发 GitHub Actions 的 mac-quick-connect 打包，"
                         "再用 --mac-launcher 指定下载产物里的 Contents/MacOS/launcher。")
    # Start clean so replaced tools, such as 1.Codex接入配置, do not linger.
    shutil.rmtree(OUTPUT_ROOT, ignore_errors=True)
    if mac_launcher is None and not options.terminal_mac:
        mac_launcher = check_mac_launcher(quick_connect.build_mac(OUTPUT_ROOT / ".mac-launcher" / "launcher"))
    runner = quick_connect.MAC_RUNNER.read_text()

    assistant_mac(MAC / f"{ASSISTANT_TOOL}.app", mac_launcher, runner)
    # Website download: fixed English name; the site's headers give the
    # Chinese file name. The Mac .app in the zip has no tool number.
    staging = OUTPUT_ROOT / ".download-staging"
    zip_application(assistant_mac(staging / f"{ASSISTANT}.app", mac_launcher, runner), DOWNLOAD / MAC_DOWNLOAD)
    shutil.rmtree(staging)
    shutil.rmtree(OUTPUT_ROOT / ".mac-launcher", ignore_errors=True)
    if options.mac_only:
        write_checksums(OUTPUT_ROOT)
        print(OUTPUT_ROOT)
        return

    quick_connect.build_windows(WINDOWS / f"{ASSISTANT_TOOL}.exe")
    shutil.copyfile(WINDOWS / f"{ASSISTANT_TOOL}.exe", DOWNLOAD / WINDOWS_DOWNLOAD)

    for title, script, identifier in TERMINAL_TOOLS:
        powershell = (PROJECT_ROOT / "install" / f"{script}.ps1").read_bytes()
        # Downloaded scripts must also parse correctly in Windows PowerShell 5.1.
        if not powershell.startswith(b"\xef\xbb\xbf"):
            powershell = b"\xef\xbb\xbf" + powershell
        windows_executable(WINDOWS / f"{title}.exe", {
            "start.ps1": powershell_runner("    & (Join-Path $PSScriptRoot 'install.ps1')"),
            "install.ps1": powershell,
        })
        mac_application(MAC / f"{title}.app", identifier, "1.0.0", {
            "install.sh": (PROJECT_ROOT / "install" / f"{script}.sh").read_bytes(),
        })


    write_checksums(OUTPUT_ROOT)
    print(OUTPUT_ROOT)


if __name__ == "__main__":
    main()
