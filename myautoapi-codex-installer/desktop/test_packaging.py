"""No credentials, real user configuration or client termination is used here."""

from __future__ import annotations

import os
import plistlib
import shlex
import shutil
import struct
import subprocess
import sys
import tempfile
import unittest
import zipfile
from pathlib import Path

import packaging

PROJECT_ROOT = Path(__file__).resolve().parents[1]
POWERSHELL = Path(os.environ.get("PWSH", PROJECT_ROOT / "codex-domestic-models/.dev-tools/pwsh/pwsh"))
ASSISTANT_RUNNER = PROJECT_ROOT / "quick-connect/macapp/run.command"
# The macOS runner of GitHub Actions builds only the Mac assistant.
MAC_ONLY = os.environ.get("MYAUTOAPI_MAC_ONLY") == "1"


class DesktopPackagingTests(unittest.TestCase):
    def test_powershell_wrapper_preserves_interactive_input_and_reports_failures(self) -> None:
        if not POWERSHELL.is_file():
            self.skipTest("本机未安装开发用 PowerShell")
        with tempfile.TemporaryDirectory(dir=packaging.WORK_ROOT) as temporary:
            root = Path(temporary) / "中文 launcher's $(literal)"
            root.mkdir()
            wrapper = root / "start.ps1"
            wrapper.write_bytes(packaging.powershell_runner("    & (Join-Path $PSScriptRoot 'install.ps1')"))
            installer = root / "install.ps1"
            installer.write_text("$value = Read-Host '输入测试值'\nif ($value -ne 'synthetic-input') { throw 'input-lost' }\nWrite-Host 'fixture-ok'\n", encoding="utf-8-sig")
            environment = packaging.build_environment()
            result = subprocess.run([str(POWERSHELL), "-NoProfile", "-File", str(wrapper)], input="synthetic-input\n", text=True, capture_output=True, env=environment)
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertIn("fixture-ok", result.stdout)
            installer.write_text("throw 'fixture-failure'", encoding="utf-8-sig")
            failed = subprocess.run([str(POWERSHELL), "-NoProfile", "-File", str(wrapper)], text=True, capture_output=True, env=environment)
            self.assertEqual(failed.returncode, 1)
            self.assertIn("fixture-failure", failed.stderr)

    def test_terminal_runner_preserves_input_and_failure_status_in_quoted_paths(self) -> None:
        if shutil.which("script") is None:
            self.skipTest("缺少伪终端工具")
        with tempfile.TemporaryDirectory(dir=packaging.WORK_ROOT) as temporary:
            root = Path(temporary) / "中文 launcher's $(literal)"
            root.mkdir()
            (root / "run.command").write_text(packaging.TERMINAL_RUNNER)
            (root / "install.sh").write_text("#!/bin/bash\nIFS= read -r value\n[ \"$value\" = synthetic-input ] || exit 91\nprintf 'fixture-ok\\n'\nexit 7\n")
            result = subprocess.run(
                ["script", "-qefc", "/bin/bash " + shlex.quote(str(root / "run.command")), str(root / "terminal.log")],
                input="synthetic-input\n\n", text=True, capture_output=True, timeout=15,
            )
            self.assertEqual(result.returncode, 7, result.stdout + result.stderr)
            self.assertIn("fixture-ok", result.stdout)
            self.assertIn("退出码：7", result.stdout)
            self.assertIn("按回车结束", result.stdout)

    def test_only_the_assistant_runner_reads_the_clipboard(self) -> None:
        self.assertIn('/bin/bash "$resource_dir/install.sh"\nresult=$?', packaging.TERMINAL_RUNNER)
        self.assertIn('/bin/bash "$resource_dir/install.sh" --from-app\n', ASSISTANT_RUNNER.read_text())

    def test_zip_application_keeps_folder_and_execute_bits(self) -> None:
        with tempfile.TemporaryDirectory(dir=packaging.WORK_ROOT) as temporary:
            app = Path(temporary) / "测试.app"
            (app / "Contents/MacOS").mkdir(parents=True)
            (app / "Contents/MacOS/launcher").write_bytes(b"binary")
            (app / "Contents/MacOS/launcher").chmod(0o755)
            (app / "Contents/Info.plist").write_bytes(b"plist")
            (app / "Contents/Info.plist").chmod(0o644)
            archive_path = packaging.zip_application(app, Path(temporary) / "out.zip")
            with zipfile.ZipFile(archive_path) as archive:
                modes = {info.filename: info.external_attr >> 16 for info in archive.infolist()}
            self.assertEqual(modes["测试.app/"] & 0o170000, 0o040000)
            self.assertEqual(modes["测试.app/Contents/MacOS/launcher"], 0o100755)
            self.assertEqual(modes["测试.app/Contents/Info.plist"], 0o100644)

    @unittest.skipIf(MAC_ONLY, "Mac 打包只生成快速接入助手")
    def test_actual_artifacts_have_correct_architecture_and_original_payloads(self) -> None:
        output = PROJECT_ROOT / "dist/desktop"
        for title, script in (("3.Codex国产模型导入", "codex-models"),):
            executable = output / "Windows-x64" / f"{title}.exe"
            data = executable.read_bytes()
            self.assertEqual(data[:2], b"MZ")
            pe_offset = struct.unpack_from("<I", data, 0x3C)[0]
            self.assertEqual(data[pe_offset:pe_offset + 4], b"PE\0\0")
            self.assertEqual(struct.unpack_from("<H", data, pe_offset + 4)[0], 0x8664)
            self.assertEqual(struct.unpack_from("<H", data, pe_offset + 24 + 68)[0], 3, "双击必须打开控制台窗口")
            with zipfile.ZipFile(executable) as archive:
                self.assertEqual(set(archive.namelist()), {"start.ps1", "install.ps1"})
                original = (PROJECT_ROOT / "install" / f"{script}.ps1").read_bytes().removeprefix(b"\xef\xbb\xbf")
                self.assertEqual(archive.read("install.ps1").removeprefix(b"\xef\xbb\xbf"), original)
            app = output / "Mac-Apple芯片" / f"{title}.app/Contents"
            info = plistlib.loads((app / "Info.plist").read_bytes())
            self.assertEqual(info["CFBundleDisplayName"], title)
            launcher = app / "MacOS" / info["CFBundleExecutable"]
            self.assertEqual(struct.unpack_from("<II", launcher.read_bytes()), (0xFEEDFACF, 0x0100000C))
            self.assertTrue(os.access(launcher, os.X_OK))
            self.assertEqual((app / "Resources/install.sh").read_bytes(), (PROJECT_ROOT / "install" / f"{script}.sh").read_bytes())
            self.assertEqual((app / "Resources/run.command").read_text(), packaging.TERMINAL_RUNNER)

    @unittest.skipIf(MAC_ONLY, "Mac 打包只生成快速接入助手")
    def test_assistant_windows_artifacts(self) -> None:
        output = PROJECT_ROOT / "dist/desktop"
        title = "1.MyAutoAPI中转快速接入助手"
        self.assertFalse((output / "Windows-x64/1.Codex接入配置.exe").exists())
        self.assertFalse((output / "Mac-Apple芯片/1.Codex接入配置.app").exists())
        executable = output / "Windows-x64" / f"{title}.exe"
        data = executable.read_bytes()
        self.assertLessEqual(len(data), 25 * 1024 * 1024)
        pe_offset = struct.unpack_from("<I", data, 0x3C)[0]
        self.assertEqual(data[pe_offset:pe_offset + 4], b"PE\0\0")
        self.assertEqual(struct.unpack_from("<H", data, pe_offset + 4)[0], 0x8664)
        self.assertEqual(struct.unpack_from("<H", data, pe_offset + 24 + 68)[0], 2, "助手是窗口程序，不打开控制台")
        self.assertEqual((output / "download/myautoapi-quick-connect-windows-x64.exe").read_bytes(), data)

    def test_assistant_mac_artifacts(self) -> None:
        output = PROJECT_ROOT / "dist/desktop"
        title = "1.MyAutoAPI中转快速接入助手"
        launchers = []

        def check_app(contents: Path, name: str) -> None:
            info = plistlib.loads((contents / "Info.plist").read_bytes())
            self.assertEqual(info["CFBundleDisplayName"], name)
            self.assertEqual(info["CFBundleIdentifier"], "com.myautoapi.codex-config")
            self.assertEqual(info["CFBundleShortVersionString"], "1.1.4")
            if info.get("MyAutoAPILauncherMode") == "terminal":
                self.assertNotIn("CFBundleURLTypes", info)
            else:
                self.assertEqual(info["CFBundleURLTypes"], [{"CFBundleURLName": "com.myautoapi.import", "CFBundleURLSchemes": ["myautoapi"]}])
            self.assertIs(info["LSUIElement"], True)
            self.assertEqual((contents / "Resources/install.sh").read_bytes(), (PROJECT_ROOT / "install/codex.sh").read_bytes())
            self.assertEqual((contents / "Resources/run.command").read_bytes(), ASSISTANT_RUNNER.read_bytes())
            self.assertEqual(sorted(path.name for path in (contents / "Resources").iterdir()), ["install.sh", "run.command"])
            launcher = contents / "MacOS" / info["CFBundleExecutable"]
            self.assertEqual(struct.unpack_from("<II", launcher.read_bytes()), (0xFEEDFACF, 0x0100000C))
            if info.get("MyAutoAPILauncherMode") == "terminal":
                self.assertIn(b"run.command", launcher.read_bytes())
            else:
                self.assertIn(b"myautoapi-quick-connect/macapp", launcher.read_bytes())
            for path in (launcher, contents / "Resources/run.command", contents / "Resources/install.sh"):
                self.assertTrue(os.access(path, os.X_OK), path)
            if sys.platform == "darwin":
                subprocess.run(["/usr/bin/plutil", "-lint", str(contents / "Info.plist")], check=True)
                # Apple silicon only runs signed code: the linker's ad-hoc
                # signature must be valid. Checked on a copy, because inside
                # the (unsigned) bundle codesign would check the bundle.
                with tempfile.TemporaryDirectory(dir=packaging.WORK_ROOT) as loose:
                    copy = Path(loose) / "launcher"
                    shutil.copyfile(launcher, copy)
                    result = subprocess.run(["/usr/bin/codesign", "--verify", "--verbose=2", str(copy)], capture_output=True, text=True)
                    self.assertEqual(result.returncode, 0, result.stderr)
                    shown = subprocess.run(["/usr/bin/codesign", "--display", "--verbose=2", str(copy)], capture_output=True, text=True)
                    self.assertIn("Signature=adhoc", shown.stderr)
            launchers.append(launcher.read_bytes())

        check_app(output / "Mac-Apple芯片" / f"{title}.app/Contents", title)
        download = output / "download/myautoapi-quick-connect-mac-arm64.zip"
        self.assertLessEqual(download.stat().st_size, 25 * 1024 * 1024)
        with tempfile.TemporaryDirectory(dir=packaging.WORK_ROOT) as temporary:
            with zipfile.ZipFile(download) as archive:
                names = archive.namelist()
                self.assertTrue(all(name.startswith("MyAutoAPI中转快速接入助手.app/") for name in names), names)
                for info in archive.infolist():
                    target = Path(temporary) / info.filename
                    if info.is_dir():
                        target.mkdir(parents=True, exist_ok=True)
                        continue
                    target.parent.mkdir(parents=True, exist_ok=True)
                    target.write_bytes(archive.read(info))
                    target.chmod((info.external_attr >> 16) & 0o777)
            check_app(Path(temporary) / "MyAutoAPI中转快速接入助手.app/Contents", "MyAutoAPI中转快速接入助手")
        self.assertEqual(launchers[0], launchers[1])

if __name__ == "__main__":
    packaging.WORK_ROOT.mkdir(parents=True, exist_ok=True)
    unittest.main()
