"""V0.8.1 native artifact and online installer contracts."""

from __future__ import annotations

import importlib.util
import os
import re
import shlex
import shutil
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path


PROJECT_ROOT = Path(__file__).resolve().parents[1]
RELEASE_DIR = PROJECT_ROOT / "release"
BUILDER_PATH = RELEASE_DIR / "build_release.py"
MODULE_SPEC = importlib.util.spec_from_file_location("myautoapi_release", BUILDER_PATH)
assert MODULE_SPEC is not None and MODULE_SPEC.loader is not None
release = importlib.util.module_from_spec(MODULE_SPEC)
sys.modules[MODULE_SPEC.name] = release
MODULE_SPEC.loader.exec_module(release)


class ReleaseContractTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.bash_installer = (RELEASE_DIR / "install.sh").read_text(encoding="utf-8")
        cls.powershell_installer = (RELEASE_DIR / "install.ps1").read_text(encoding="utf-8")

    def run_linux_entry(self, temporary_root: Path, *, damage_binary: bool = False, key_state: str = "existing") -> None:
        if shutil.which("script") is None:
            self.skipTest("当前环境没有 script，无法建立伪终端")
        artifact = release.artifact_name("linux", "amd64")
        native = PROJECT_ROOT / "dist" / release.RELEASE_REVISION / artifact
        codex_home = temporary_root / "codex home"
        environment = os.environ.copy()
        environment["CODEX_HOME"] = str(codex_home)
        initial = subprocess.run([str(native), "install"], input="synthetic-original-key\n", text=True, capture_output=True, env=environment)
        self.assertEqual(initial.returncode, 0, initial.stderr)
        tool_dir = codex_home / "tools/myautoapi-imagines-tool"
        key_path = tool_dir / "api-key"
        if key_state == "missing":
            key_path.unlink()
        elif key_state == "empty":
            key_path.write_bytes(b"")
        elif key_state == "whitespace":
            key_path.write_bytes(b" \n\t")
        if damage_binary:
            shutil.copyfile("/usr/bin/true", tool_dir / "myautoapi-imagines-tool")
        mock_bin = temporary_root / "bin"
        mock_bin.mkdir()
        download_log = temporary_root / "downloaded"
        mocked_curl = mock_bin / "curl"
        mocked_curl.write_text(
            "#!/usr/bin/env bash\n"
            "for ((i=1; i <= $#; i++)); do\n"
            "  if [[ ${!i} == --output ]]; then\n"
            "    next=$((i+1)); cp \"$TEST_NATIVE\" \"${!next}\"\n"
            "    printf 'downloaded' > \"$TEST_DOWNLOAD_LOG\"; exit 0\n"
            "  fi\n"
            "done\nexit 91\n"
        )
        mocked_curl.chmod(0o755)
        environment.update({"PATH": f"{mock_bin}:" + environment["PATH"], "TEST_NATIVE": str(native), "TEST_DOWNLOAD_LOG": str(download_log)})
        command = "cat " + shlex.quote(str(RELEASE_DIR / "install.sh")) + " | bash"
        completed = subprocess.run(
            ["script", "-qefc", "bash -c " + shlex.quote(command), str(temporary_root / "terminal.log")],
            input="synthetic-new-key\n", text=True, capture_output=True, env=environment, timeout=30,
        )
        self.assertEqual(completed.returncode, 0, completed.stdout + completed.stderr)
        self.assertIn("请输入专门画图分组 Key", completed.stdout)
        self.assertEqual(key_path.read_text(), "synthetic-new-key")
        self.assertEqual(download_log.exists(), damage_binary)
        self.assertEqual(release.sha256(tool_dir / "myautoapi-imagines-tool"), release.sha256(native))

    def test_cached_same_version_always_prompts_and_replaces_key(self) -> None:
        for state in ("existing", "missing", "empty", "whitespace"):
            with self.subTest(key_state=state), tempfile.TemporaryDirectory() as temporary:
                self.run_linux_entry(Path(temporary), key_state=state)

    def test_cached_wrong_program_is_replaced_before_execution(self) -> None:
        with tempfile.TemporaryDirectory() as temporary:
            self.run_linux_entry(Path(temporary), damage_binary=True)

    def test_builder_defines_all_supported_native_targets(self) -> None:
        self.assertEqual(release.RELEASE_REVISION, "v0.8.1")
        self.assertEqual(
            set(release.TARGETS),
            {
                ("windows", "amd64"),
                ("darwin", "arm64"),
                ("darwin", "amd64"),
                ("linux", "amd64"),
            },
        )
        self.assertTrue(release.artifact_name("windows", "amd64").endswith(".exe"))
        self.assertFalse(release.artifact_name("darwin", "arm64").endswith(".exe"))
        self.assertFalse(release.artifact_name("linux", "amd64").endswith(".exe"))

    def test_online_entries_pin_v081_artifacts_and_hashes(self) -> None:
        for script in (
            self.bash_installer,
            self.powershell_installer,
        ):
            self.assertIn("releases/v0.8.1", script)
            self.assertNotIn("v0.2-r1.zip", script)
            self.assertNotRegex(script, r"(?i)python|安装\.py|Expand-Archive")
        bash_hashes = re.findall(r'expected_sha256="([0-9a-f]{64})"', self.bash_installer)
        powershell_hashes = set(
            re.findall(r'\$ExpectedSha256 = "([0-9a-f]{64})"', self.powershell_installer)
        )
        self.assertEqual(len(bash_hashes), 3)
        self.assertEqual(len(powershell_hashes), 1)
        for digest in set(bash_hashes) | powershell_hashes:
            self.assertEqual(len(digest), 64)

    def test_built_artifacts_match_checksums_and_online_entries(self) -> None:
        output_dir = PROJECT_ROOT / "dist" / release.RELEASE_REVISION
        checksum_path = output_dir / "SHA256SUMS"
        self.assertTrue(checksum_path.is_file(), "请先运行 python3 release/build_release.py")
        listed_hashes = {
            artifact: digest
            for digest, artifact in (
                line.split(maxsplit=1)
                for line in checksum_path.read_text(encoding="ascii").splitlines()
            )
        }
        expected_artifacts = {
            release.artifact_name(goos, goarch)
            for goos, goarch in release.TARGETS
        }
        self.assertEqual(set(listed_hashes), expected_artifacts)

        for artifact, expected_digest in listed_hashes.items():
            artifact_path = output_dir / artifact
            self.assertTrue(artifact_path.is_file())
            self.assertEqual(release.sha256(artifact_path), expected_digest)
            if "-windows-" in artifact:
                self.assertIn(f'$Artifact = "{artifact}"', self.powershell_installer)
                self.assertIn(f'$ExpectedSha256 = "{expected_digest}"', self.powershell_installer)
            else:
                self.assertIn(f'artifact="{artifact}"', self.bash_installer)
                self.assertIn(f'expected_sha256="{expected_digest}"', self.bash_installer)

    def test_online_entries_verify_before_running_native_program(self) -> None:
        self.assertLess(self.bash_installer.index("actual_sha256"), self.bash_installer.index('"$binary_path" install'))
        self.assertLess(self.powershell_installer.index("Get-FileHash"), self.powershell_installer.index("& $BinaryPath install"))
        self.assertIn("--proto-redir '=https'", self.bash_installer)

    def test_piped_bash_installer_reads_key_from_terminal(self) -> None:
        """`curl | bash` leaves stdin at EOF, so the native installer must use /dev/tty."""
        if shutil.which("script") is None:
            self.skipTest("当前环境没有 script，无法建立伪终端")

        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_root = Path(temporary_directory)
            mock_bin = temporary_root / "bin"
            mock_bin.mkdir()
            expected_sha256 = re.search(r'expected_sha256="([0-9a-f]{64})"', self.bash_installer).group(1)
            result_path = temporary_root / "result"
            native_program = temporary_root / "native-program"
            native_program.write_text(
                "#!/usr/bin/env bash\n"
                "if [[ ! -t 0 ]]; then exit 71; fi\n"
                "IFS= read -r ignored || exit 72\n"
                "printf 'terminal-input-ok\\n' > \"$TEST_RESULT\"\n",
                encoding="utf-8",
            )
            native_program.chmod(0o755)
            (mock_bin / "uname").write_text(
                "#!/usr/bin/env bash\n"
                "if [[ \"$1\" == \"-s\" ]]; then printf 'Darwin\\n'; else printf 'arm64\\n'; fi\n",
                encoding="utf-8",
            )
            (mock_bin / "curl").write_text(
                "#!/usr/bin/env bash\n"
                "for ((index = 1; index <= $#; index++)); do\n"
                "  if [[ \"${!index}\" == \"--output\" ]]; then\n"
                "    next=$((index + 1)); cp \"$TEST_NATIVE_PROGRAM\" \"${!next}\"; exit 0\n"
                "  fi\n"
                "done\n"
                "exit 73\n",
                encoding="utf-8",
            )
            (mock_bin / "sha256sum").write_text(
                "#!/usr/bin/env bash\n"
                f"printf '%s  %s\\n' '{expected_sha256}' \"$1\"\n",
                encoding="utf-8",
            )
            (mock_bin / "file").write_text(
                "#!/usr/bin/env bash\n"
                "printf 'Mach-O 64-bit arm64 executable\\n'\n",
                encoding="utf-8",
            )
            for mock in mock_bin.iterdir():
                mock.chmod(0o755)

            environment = {
                "PATH": f"{mock_bin}:{Path('/usr/bin')}:{Path('/bin')}",
                "TEST_NATIVE_PROGRAM": str(native_program),
                "TEST_RESULT": str(result_path),
                "CODEX_HOME": str(temporary_root / "codex-home"),
            }
            shell_command = f"cat {RELEASE_DIR / 'install.sh'} | bash"
            completed = subprocess.run(
                [
                    "script",
                    "-qefc",
                    "env "
                    + " ".join(f"{key}={value}" for key, value in environment.items())
                    + f" bash -c {shell_command!r}",
                    str(temporary_root / "terminal.log"),
                ],
                input="test-terminal-input\n",
                text=True,
                capture_output=True,
                check=False,
            )
            self.assertEqual(completed.returncode, 0, completed.stderr)
            self.assertEqual(result_path.read_text(encoding="utf-8"), "terminal-input-ok\n")

    def test_install_state_version_is_v081(self) -> None:
        self.assertEqual((PROJECT_ROOT / "VERSION").read_text(encoding="utf-8").strip(), "0.8.1")

    def test_powershell_entry_points_do_not_exit_the_calling_session(self) -> None:
        for script in (self.powershell_installer,):
            self.assertNotRegex(script, r"(?im)^\s*exit\b")

    def test_powershell_entry_points_propagate_failures(self) -> None:
        for script in (self.powershell_installer,):
            self.assertRegex(script, r'(?s)if\s*\(\$Architecture.*?-ne\s*"AMD64"\).*?throw\s*\(')
            self.assertRegex(script, r"(?s)catch\s*\{\s*throw\s*\(")

    def test_powershell_download_shows_percent_without_byte_progress(self) -> None:
        for script in (self.powershell_installer,):
            self.assertIn("Download-WithPercent", script)
            self.assertIn("Write-Progress", script)
            self.assertIn("-PercentComplete $Percent", script)
            self.assertNotIn("Invoke-WebRequest", script)

    def test_release_scripts_reference_all_supported_packages(self) -> None:
        combined = "\n".join(
            (self.bash_installer, self.powershell_installer)
        )
        self.assertIn("windows-amd64.exe", combined)
        self.assertIn("darwin-arm64", combined)
        self.assertIn("darwin-amd64", combined)
        self.assertIn("linux-amd64", combined)
        for unsupported in ("windows-arm64", "linux-arm64"):
            self.assertNotIn(unsupported, combined)

    def test_macos_installer_selects_native_and_rosetta_architectures(self) -> None:
        self.assertIn("Darwin/x86_64", self.bash_installer)
        self.assertIn("darwin-amd64", self.bash_installer)
        self.assertIn("sysctl.proc_translated", self.bash_installer)
        self.assertIn("Darwin/x86_64/1", self.bash_installer)
        self.assertIn("expected_binary_format", self.bash_installer)

    def test_shell_installer_downloads_the_expected_artifact_for_each_platform(self) -> None:
        cases = (
            ("Darwin", "arm64", "0", "darwin-arm64", "Mach-O 64-bit executable arm64"),
            ("Darwin", "x86_64", "0", "darwin-amd64", "Mach-O 64-bit executable x86_64"),
            ("Darwin", "x86_64", "1", "darwin-arm64", "Mach-O 64-bit executable arm64"),
            ("Linux", "x86_64", "0", "linux-amd64", "ELF 64-bit LSB executable, x86-64"),
        )
        hashes = dict(
            re.findall(
                r'artifact="([^"]+)"\n\s*expected_sha256="([0-9a-f]{64})"',
                self.bash_installer,
            )
        )

        for system, machine, translated, artifact_suffix, file_description in cases:
            with self.subTest(system=system, machine=machine, translated=translated):
                with tempfile.TemporaryDirectory() as temporary_directory:
                    temporary_root = Path(temporary_directory)
                    mock_bin = temporary_root / "bin"
                    mock_bin.mkdir()
                    download_log = temporary_root / "download-url"
                    artifact = next(name for name in hashes if name.endswith(artifact_suffix))

                    (mock_bin / "uname").write_text(
                        "#!/usr/bin/env bash\n"
                        f"if [[ \"$1\" == \"-s\" ]]; then printf '{system}\\n'; else printf '{machine}\\n'; fi\n",
                        encoding="utf-8",
                    )
                    (mock_bin / "sysctl").write_text(
                        "#!/usr/bin/env bash\n"
                        f"printf '{translated}\\n'\n",
                        encoding="utf-8",
                    )
                    (mock_bin / "curl").write_text(
                        "#!/usr/bin/env bash\n"
                        "for ((index = 1; index <= $#; index++)); do\n"
                        "  value=${!index}\n"
                        "  if [[ \"$value\" == https://* ]]; then printf '%s\\n' \"$value\" > \"$TEST_DOWNLOAD_LOG\"; fi\n"
                        "  if [[ \"$value\" == \"--output\" ]]; then next=$((index + 1)); printf 'fake-mach-o' > \"${!next}\"; fi\n"
                        "done\n",
                        encoding="utf-8",
                    )
                    (mock_bin / "sha256sum").write_text(
                        "#!/usr/bin/env bash\n"
                        "printf '%s  %s\\n' \"$TEST_EXPECTED_SHA256\" \"$1\"\n",
                        encoding="utf-8",
                    )
                    (mock_bin / "file").write_text(
                        "#!/usr/bin/env bash\n"
                        f"printf '{file_description}\\n'\n",
                        encoding="utf-8",
                    )
                    for mock_path in mock_bin.iterdir():
                        mock_path.chmod(0o755)

                    environment = os.environ.copy()
                    environment.update(
                        {
                            "PATH": f"{mock_bin}:/usr/bin:/bin",
                            "HOME": str(temporary_root / "home"),
                            "TEST_DOWNLOAD_LOG": str(download_log),
                            "TEST_EXPECTED_SHA256": hashes[artifact],
                        }
                    )
                    completed = subprocess.run(
                        ["bash", str(RELEASE_DIR / "install.sh")],
                        text=True,
                        capture_output=True,
                        check=False,
                        env=environment,
                    )

                    self.assertNotEqual(completed.returncode, 0)
                    self.assertEqual(
                        download_log.read_text(encoding="utf-8").strip(),
                        f"https://docs.myautoapi.com/install/myautoapi-imagines-tool/releases/v0.8.1/{artifact}",
                    )

    def test_shell_installer_repairs_same_version_linux_binary_without_execute_permission(self) -> None:
        if shutil.which("script") is None:
            self.skipTest("当前环境没有 script，无法建立伪终端")

        linux_release = re.search(
            r'artifact="([^"]*linux-amd64)"\n\s*expected_sha256="([0-9a-f]{64})"',
            self.bash_installer,
        )
        assert linux_release is not None
        artifact, expected_sha256 = linux_release.groups()

        with tempfile.TemporaryDirectory() as temporary_directory:
            temporary_root = Path(temporary_directory)
            mock_bin = temporary_root / "bin"
            mock_bin.mkdir()
            codex_home = temporary_root / "codex-home"
            installed_tool_dir = codex_home / "tools" / "myautoapi-imagines-tool"
            installed_tool_dir.mkdir(parents=True)
            installed_binary = installed_tool_dir / "myautoapi-imagines-tool"
            installed_binary.write_text("stale installed binary\n", encoding="utf-8")
            installed_binary.chmod(0o644)
            (installed_tool_dir / "RELEASE").write_text("V0.8.1\n", encoding="utf-8")

            download_log = temporary_root / "download-url"
            result_path = temporary_root / "result"
            native_program = temporary_root / "native-program"
            native_program.write_text(
                "#!/usr/bin/env bash\n"
                "[[ -x \"$0\" ]] || exit 81\n"
                "[[ \"$1\" == \"install\" ]] || exit 82\n"
                "printf 'repaired-and-launched\\n' > \"$TEST_RESULT\"\n",
                encoding="utf-8",
            )
            native_program.chmod(0o644)

            (mock_bin / "uname").write_text(
                "#!/usr/bin/env bash\n"
                "if [[ \"$1\" == \"-s\" ]]; then printf 'Linux\\n'; else printf 'x86_64\\n'; fi\n",
                encoding="utf-8",
            )
            (mock_bin / "curl").write_text(
                "#!/usr/bin/env bash\n"
                "for ((index = 1; index <= $#; index++)); do\n"
                "  value=${!index}\n"
                "  if [[ \"$value\" == https://* ]]; then printf '%s\\n' \"$value\" > \"$TEST_DOWNLOAD_LOG\"; fi\n"
                "  if [[ \"$value\" == \"--output\" ]]; then next=$((index + 1)); cp \"$TEST_NATIVE_PROGRAM\" \"${!next}\"; fi\n"
                "done\n",
                encoding="utf-8",
            )
            (mock_bin / "sha256sum").write_text(
                "#!/usr/bin/env bash\n"
                "printf '%s  %s\\n' \"$TEST_EXPECTED_SHA256\" \"$1\"\n",
                encoding="utf-8",
            )
            (mock_bin / "file").write_text(
                "#!/usr/bin/env bash\n"
                "printf 'ELF 64-bit LSB executable, x86-64\\n'\n",
                encoding="utf-8",
            )
            for mock_path in mock_bin.iterdir():
                mock_path.chmod(0o755)

            environment = {
                "PATH": f"{mock_bin}:/usr/bin:/bin",
                "CODEX_HOME": str(codex_home),
                "TEST_DOWNLOAD_LOG": str(download_log),
                "TEST_EXPECTED_SHA256": expected_sha256,
                "TEST_NATIVE_PROGRAM": str(native_program),
                "TEST_RESULT": str(result_path),
            }
            shell_command = "env " + " ".join(
                f"{key}={shlex.quote(value)}" for key, value in environment.items()
            ) + f" bash {shlex.quote(str(RELEASE_DIR / 'install.sh'))}"
            completed = subprocess.run(
                ["script", "-qefc", shell_command, "/dev/null"],
                text=True,
                capture_output=True,
                check=False,
            )

            self.assertTrue(
                download_log.is_file(),
                "同版本但缺少执行权限时仍应重新下载正式制品",
            )
            self.assertEqual(completed.returncode, 0, completed.stderr)
            self.assertEqual(result_path.read_text(encoding="utf-8"), "repaired-and-launched\n")
            self.assertEqual(
                download_log.read_text(encoding="utf-8").strip(),
                f"https://docs.myautoapi.com/install/myautoapi-imagines-tool/releases/v0.8.1/{artifact}",
            )


if __name__ == "__main__":
    unittest.main()
