"""V0.8.1 installer contract tests without real credentials or network calls."""

from __future__ import annotations

import contextlib
import hashlib
import importlib.util
import io
import json
import shutil
import sys
import tempfile
import unittest
from unittest import mock
from pathlib import Path


PROJECT_ROOT = Path(__file__).resolve().parents[1]
INSTALLER_PATH = PROJECT_ROOT / "installer" / "安装.py"
MODULE_SPEC = importlib.util.spec_from_file_location("myautoapi_installer", INSTALLER_PATH)
assert MODULE_SPEC is not None and MODULE_SPEC.loader is not None
installer = importlib.util.module_from_spec(MODULE_SPEC)
sys.modules[MODULE_SPEC.name] = installer
MODULE_SPEC.loader.exec_module(installer)


FAKE_AUTH_KEY = "not-a-real-auth-key-never-print-or-copy"
FAKE_DEDICATED_KEY = "not-a-real-dedicated-key-for-tests"
FAKE_LEGACY_KEY = "legacy-key-never-print-or-copy"


class InstallerContractTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary_directory = tempfile.TemporaryDirectory()
        self.addCleanup(self.temporary_directory.cleanup)
        self.root = Path(self.temporary_directory.name)
        self.codex_home = self.root / "用户目录" / ".codex-custom"
        self.environment = {"CODEX_HOME": str(self.codex_home)}
        self.output = io.StringIO()

    @property
    def tool_dir(self) -> Path:
        return self.codex_home / "tools" / installer.TOOL_NAME

    @property
    def skill_dir(self) -> Path:
        return self.codex_home / "skills" / installer.SKILL_NAME

    @property
    def auth_path(self) -> Path:
        return self.codex_home / "auth.json"

    def prepare_auth(self) -> None:
        self.codex_home.mkdir(parents=True, exist_ok=True)
        self.auth_path.write_text(
            json.dumps({"OPENAI_API_KEY": FAKE_AUTH_KEY}),
            encoding="utf-8",
        )

    def auth_digest(self) -> str:
        digest = hashlib.sha256()
        with self.auth_path.open("rb") as handle:
            for chunk in iter(lambda: handle.read(1024 * 1024), b""):
                digest.update(chunk)
        return digest.hexdigest()

    def assert_auth_unchanged(self, before: str) -> None:
        self.assertTrue(self.auth_path.is_file())
        self.assertEqual(self.auth_digest(), before)

    def install(self, **kwargs):
        kwargs.setdefault("input_stream", io.StringIO(FAKE_DEDICATED_KEY + "\n"))
        return installer.install(
            source_root=PROJECT_ROOT,
            environ=self.environment,
            output=self.output,
            **kwargs,
        )

    def uninstall(self, **kwargs):
        return installer.uninstall(
            source_root=PROJECT_ROOT,
            environ=self.environment,
            output=self.output,
            **kwargs,
        )

    def test_first_install_writes_dedicated_key_without_copying_auth(self) -> None:
        self.prepare_auth()
        before = self.auth_digest()

        result = self.install()

        self.assertEqual(result.status, "installed")
        self.assertTrue((self.tool_dir / "生成图片.py").is_file())
        self.assertTrue((self.tool_dir / "VERSION").is_file())
        self.assertTrue((self.tool_dir / "install-manifest.json").is_file())
        self.assertTrue((self.skill_dir / "SKILL.md").is_file())
        self.assertTrue((self.skill_dir / "agents" / "openai.yaml").is_file())
        key_path = self.tool_dir / installer.LEGACY_KEY_FILE_NAME
        self.assertTrue(key_path.is_file())
        self.assertGreater(key_path.stat().st_size, 0)
        self.assertTrue((self.tool_dir / installer.KEY_SOURCE_FILE_NAME).is_file())
        self.assertIn(installer.RULE_START, (self.codex_home / "AGENTS.md").read_text())
        self.assertIn("专门画图分组", self.output.getvalue())
        self.assertNotIn(FAKE_AUTH_KEY, self.output.getvalue())
        self.assertNotIn(FAKE_DEDICATED_KEY, self.output.getvalue())
        self.assert_auth_unchanged(before)
        self.assert_auth_not_copied()

    def test_install_does_not_require_auth_json(self) -> None:
        result = self.install()

        self.assertEqual(result.status, "installed")
        self.assertFalse(self.auth_path.exists())
        self.assertTrue((self.tool_dir / installer.LEGACY_KEY_FILE_NAME).is_file())

    def test_repeated_install_replaces_key_without_printing_it(self) -> None:
        self.install()
        key_path = self.tool_dir / installer.LEGACY_KEY_FILE_NAME
        replacement = "synthetic-replacement-key"
        self.output = io.StringIO()
        self.install(input_stream=io.StringIO(replacement + "\n"))
        self.assertEqual(key_path.read_text(), replacement)
        self.assertIn("请输入专门画图分组 Key", self.output.getvalue())
        self.assertNotIn(replacement, self.output.getvalue())

    def test_repeated_install_empty_input_preserves_key(self) -> None:
        self.install()
        key_path = self.tool_dir / installer.LEGACY_KEY_FILE_NAME
        before = key_path.read_bytes()
        with self.assertRaises(installer.InstallationError):
            self.install(input_stream=io.StringIO("\n"))
        self.assertEqual(key_path.read_bytes(), before)

    def test_repeated_install_failure_restores_key(self) -> None:
        self.install()
        key_path = self.tool_dir / installer.LEGACY_KEY_FILE_NAME
        before = key_path.read_bytes()
        def fail(stage: str) -> None:
            if stage == "managed-files-written":
                raise OSError("synthetic replacement failure")
        with self.assertRaises(installer.InstallationError):
            self.install(input_stream=io.StringIO("synthetic-new-key\n"), failure_hook=fail)
        self.assertEqual(key_path.read_bytes(), before)

    def test_same_version_reconfigures_without_auth_access(self) -> None:
        self.prepare_auth()
        before = self.auth_digest()
        self.install()
        second_output = io.StringIO()

        second = installer.install(
            source_root=PROJECT_ROOT,
            environ=self.environment,
            output=second_output,
            input_stream=io.StringIO("synthetic-replacement-key\n"),
        )

        self.assertEqual(second.status, "configured")
        self.assertIn("请输入专门画图分组 Key", second_output.getvalue())
        self.assert_auth_unchanged(before)

    def test_same_version_repairs_managed_files_without_auth_access(self) -> None:
        self.prepare_auth()
        before = self.auth_digest()
        self.install()
        (self.tool_dir / "生成图片.py").write_text("damaged\n", encoding="utf-8")
        (self.skill_dir / "SKILL.md").unlink()

        repaired = self.install()

        self.assertEqual(repaired.status, "repaired")
        self.assertEqual(
            (self.tool_dir / "生成图片.py").read_bytes(),
            (PROJECT_ROOT / "tool" / "生成图片.py").read_bytes(),
        )
        self.assertEqual(
            (self.skill_dir / "SKILL.md").read_bytes(),
            (PROJECT_ROOT / "skill-template" / installer.SKILL_NAME / "SKILL.md").read_bytes(),
        )
        self.assert_auth_unchanged(before)

    def test_v075_install_upgrades_and_replaces_the_dedicated_key(self) -> None:
        self.install()
        key_path = self.tool_dir / installer.LEGACY_KEY_FILE_NAME
        key_before = key_path.read_bytes()
        state_path = self.tool_dir / installer.STATE_FILE_NAME
        state = json.loads(state_path.read_text(encoding="utf-8"))
        state["version"] = "0.7.5"
        state_path.write_text(json.dumps(state, ensure_ascii=False), encoding="utf-8")
        (self.tool_dir / "VERSION").write_text("0.7.5\n", encoding="utf-8")
        self.output = io.StringIO()

        result = self.install(input_stream=io.StringIO("synthetic-upgrade-key\n"))

        self.assertEqual(result.status, "upgraded")
        self.assertEqual(key_path.read_bytes(), b"synthetic-upgrade-key")
        self.assertEqual((self.tool_dir / "VERSION").read_text(encoding="utf-8"), "0.8.1\n")

    def test_v076_install_upgrades_to_v081_and_replaces_the_dedicated_key(self) -> None:
        self.install()
        key_path = self.tool_dir / installer.LEGACY_KEY_FILE_NAME
        key_before = key_path.read_bytes()
        state_path = self.tool_dir / installer.STATE_FILE_NAME
        state = json.loads(state_path.read_text(encoding="utf-8"))
        state["version"] = "0.7.6"
        state_path.write_text(json.dumps(state, ensure_ascii=False), encoding="utf-8")
        (self.tool_dir / "VERSION").write_text("0.7.6\n", encoding="utf-8")
        self.output = io.StringIO()

        result = self.install(input_stream=io.StringIO("synthetic-upgrade-key\n"))

        self.assertEqual(result.status, "upgraded")
        self.assertEqual(key_path.read_bytes(), b"synthetic-upgrade-key")
        self.assertEqual((self.tool_dir / "VERSION").read_text(encoding="utf-8"), "0.8.1\n")

    def test_legacy_key_is_ignored_and_preserved(self) -> None:
        self.prepare_auth()
        before = self.auth_digest()
        self.install()
        legacy_path = self.tool_dir / installer.LEGACY_KEY_FILE_NAME
        legacy_path.write_text(FAKE_LEGACY_KEY, encoding="utf-8")

        repaired = self.install()
        result = self.uninstall()

        self.assertEqual(repaired.status, "configured")
        self.assertEqual(result.status, "uninstalled")
        self.assertTrue(legacy_path.is_file())
        self.assertFalse((self.tool_dir / "生成图片.py").exists())
        self.assert_auth_unchanged(before)

    def test_unknown_target_directory_stops_without_changes(self) -> None:
        self.prepare_auth()
        before = self.auth_digest()
        self.tool_dir.mkdir(parents=True)
        unknown = self.tool_dir / "用户文件.txt"
        unknown.write_text("keep", encoding="utf-8")

        with self.assertRaises(installer.InstallationError):
            self.install()

        self.assertEqual(unknown.read_text(encoding="utf-8"), "keep")
        self.assertFalse((self.tool_dir / "生成图片.py").exists())
        self.assertFalse(self.skill_dir.exists())
        self.assert_auth_unchanged(before)

    def test_dangling_tool_directory_link_is_not_overwritten(self) -> None:
        self.prepare_auth()
        before = self.auth_digest()
        self.tool_dir.parent.mkdir(parents=True)
        try:
            self.tool_dir.symlink_to(self.root / "missing-target", target_is_directory=True)
        except OSError as error:
            self.skipTest(f"当前系统不允许创建符号链接：{error}")

        with self.assertRaisesRegex(installer.InstallationError, "符号链接"):
            self.install()

        self.assertTrue(self.tool_dir.is_symlink())
        self.assert_auth_unchanged(before)

    def test_parent_tools_or_skills_symlink_stops_without_external_write(self) -> None:
        outside = self.root / "outside"
        outside.mkdir()
        for parent_name in ("tools", "skills"):
            with self.subTest(parent_name=parent_name):
                case_home = self.root / f"symlink-{parent_name}"
                case_home.mkdir()
                try:
                    (case_home / parent_name).symlink_to(outside, target_is_directory=True)
                except OSError as error:
                    self.skipTest(f"当前系统不允许创建符号链接：{error}")
                with self.assertRaisesRegex(installer.InstallationError, "符号链接"):
                    installer.install(
                        source_root=PROJECT_ROOT,
                        environ={"CODEX_HOME": str(case_home)},
                        output=io.StringIO(),
                    )
        self.assertEqual(list(outside.iterdir()), [])

    def test_override_file_blocks_before_any_write(self) -> None:
        self.prepare_auth()
        before = self.auth_digest()
        override = self.codex_home / "AGENTS.override.md"
        override.write_text("# override\n", encoding="utf-8")

        with self.assertRaisesRegex(installer.InstallationError, "AGENTS.override.md"):
            self.install()

        self.assertFalse(self.tool_dir.exists())
        self.assertFalse(self.skill_dir.exists())
        self.assert_auth_unchanged(before)

    def test_failure_after_managed_writes_rolls_back_without_auth_access(self) -> None:
        self.prepare_auth()
        before = self.auth_digest()
        agents = self.codex_home / "AGENTS.md"
        original_agents = "# 原有内容\n"
        agents.write_text(original_agents, encoding="utf-8")

        def fail(stage: str) -> None:
            if stage == "managed-files-written":
                raise OSError("simulated copy interruption")

        with self.assertRaises(installer.InstallationError):
            self.install(failure_hook=fail)

        self.assertEqual(agents.read_text(encoding="utf-8"), original_agents)
        self.assertFalse((self.tool_dir / "生成图片.py").exists())
        self.assertFalse((self.tool_dir / installer.LEGACY_KEY_FILE_NAME).exists())
        self.assertFalse(self.skill_dir.exists())
        self.assert_auth_unchanged(before)
        self.assert_auth_not_copied()

    def test_uninstall_removes_only_managed_files_and_leaves_auth_untouched(self) -> None:
        self.prepare_auth()
        before = self.auth_digest()
        self.install()
        self.output = io.StringIO()

        result = self.uninstall()

        self.assertEqual(result.status, "uninstalled")
        self.assertFalse((self.tool_dir / "生成图片.py").exists())
        self.assertFalse(self.skill_dir.exists())
        self.assertIn("不会读取、修改或删除", self.output.getvalue())
        self.assert_auth_unchanged(before)

    def test_nonempty_override_does_not_block_explicit_uninstall(self) -> None:
        self.prepare_auth()
        before = self.auth_digest()
        self.install()
        override = self.codex_home / "AGENTS.override.md"
        override.write_text("# 用户自己的覆盖规则\n", encoding="utf-8")

        result = self.uninstall()

        self.assertEqual(result.status, "uninstalled")
        self.assertEqual(override.read_text(encoding="utf-8"), "# 用户自己的覆盖规则\n")
        self.assert_auth_unchanged(before)

    def test_uninstall_stops_for_unknown_files(self) -> None:
        self.prepare_auth()
        before = self.auth_digest()
        self.install()
        unknown = self.skill_dir / "用户文件.txt"
        unknown.write_text("keep", encoding="utf-8")

        with self.assertRaisesRegex(installer.InstallationError, "未知文件"):
            self.uninstall()

        self.assertTrue(unknown.is_file())
        self.assertTrue((self.tool_dir / "生成图片.py").is_file())
        self.assert_auth_unchanged(before)

    def test_manifest_validation_failure_happens_before_target_write(self) -> None:
        self.prepare_auth()
        before = self.auth_digest()
        package_copy = self.root / "package"
        shutil.copytree(PROJECT_ROOT, package_copy, ignore=shutil.ignore_patterns(".git", "__pycache__"))
        (package_copy / "tool" / "生成图片.py").write_text("tampered\n", encoding="utf-8")

        with self.assertRaisesRegex(installer.InstallationError, "校验"):
            installer.install(
                source_root=package_copy,
                environ=self.environment,
                output=self.output,
            )

        self.assertFalse(self.tool_dir.exists())
        self.assert_auth_unchanged(before)

    def test_cli_accepts_only_internal_integrated_flag(self) -> None:
        parser = installer.create_parser()
        for arguments in (
            ["install", "--api-key", FAKE_AUTH_KEY],
            ["install", "--replace-key"],
            ["uninstall", "--remove-key"],
        ):
            with self.subTest(arguments=arguments):
                with contextlib.redirect_stderr(io.StringIO()):
                    with self.assertRaises(SystemExit):
                        parser.parse_args(arguments)

        stdout = io.StringIO()
        with contextlib.redirect_stdout(stdout), mock.patch.object(
            sys, "stdin", io.StringIO(FAKE_DEDICATED_KEY + "\n")
        ):
            exit_code = installer.main(["install", "--source-root", str(PROJECT_ROOT)], environ=self.environment)
        self.assertEqual(exit_code, 0)
        self.assertNotIn(FAKE_AUTH_KEY, stdout.getvalue())
        self.assertNotIn(FAKE_DEDICATED_KEY, stdout.getvalue())
        self.assertTrue((self.tool_dir / installer.LEGACY_KEY_FILE_NAME).is_file())

    def assert_auth_not_copied(self) -> None:
        for path in self.codex_home.rglob("*"):
            if not path.is_file() or path == self.auth_path:
                continue
            self.assertNotIn(FAKE_AUTH_KEY.encode("utf-8"), path.read_bytes(), str(path))


if __name__ == "__main__":
    unittest.main()
