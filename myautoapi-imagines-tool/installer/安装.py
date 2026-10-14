#!/usr/bin/env python3
"""Install or uninstall the V0.8.1 MyAutoAPI image tool with Python stdlib only."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import shutil
import stat
import sys
import tempfile
import uuid
from collections.abc import Callable, Mapping
from dataclasses import dataclass
from datetime import datetime
from pathlib import Path
from typing import Any, TextIO


MANAGED_BY = "myautoapi-imagines-tool"
TOOL_NAME = "myautoapi-imagines-tool"
SKILL_NAME = "myautoapi-image-generation"
STATE_FILE_NAME = "install-manifest.json"
LEGACY_KEY_FILE_NAME = "api-key"
KEY_SOURCE_FILE_NAME = "key-source"
KEY_SOURCE_DEDICATED = "dedicated-image-key"
RULE_START = "<!-- myautoapi-image-generation:start -->"
RULE_END = "<!-- myautoapi-image-generation:end -->"
RULE_BLOCK = (Path(__file__).with_name("AGENTS.rule.md")).read_text(encoding="utf-8").strip()

PACKAGE_FILES = {
    "VERSION",
    "tool/生成图片.py",
    f"skill-template/{SKILL_NAME}/SKILL.md",
    f"skill-template/{SKILL_NAME}/agents/openai.yaml",
}


class InstallationError(Exception):
    """A non-sensitive installation or uninstallation failure."""


@dataclass(frozen=True)
class InstallResult:
    status: str
    codex_home: Path
    tool_dir: Path
    skill_dir: Path
    agents_path: Path


@dataclass(frozen=True)
class Layout:
    codex_home: Path
    tool_dir: Path
    skill_dir: Path
    agents_path: Path
    override_path: Path
    state_path: Path


FailureHook = Callable[[str], None]


def resolve_codex_home(
    environ: Mapping[str, str] | None = None, home: Path | str | None = None
) -> Path:
    environment = os.environ if environ is None else environ
    configured = environment.get("CODEX_HOME", "").strip()
    if configured:
        return Path(configured).expanduser()
    fallback_home = Path.home() if home is None else Path(home)
    return fallback_home.expanduser() / ".codex"


def build_layout(codex_home: Path) -> Layout:
    tool_dir = codex_home / "tools" / TOOL_NAME
    return Layout(
        codex_home=codex_home,
        tool_dir=tool_dir,
        skill_dir=codex_home / "skills" / SKILL_NAME,
        agents_path=codex_home / "AGENTS.md",
        override_path=codex_home / "AGENTS.override.md",
        state_path=tool_dir / STATE_FILE_NAME,
    )


def _sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def _load_json_object(path: Path, label: str) -> dict[str, Any]:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, UnicodeDecodeError, json.JSONDecodeError) as error:
        raise InstallationError(f"无法读取{label}：{path}") from error
    if not isinstance(value, dict):
        raise InstallationError(f"{label}格式无效：{path}")
    return value


def validate_package(source_root: Path) -> dict[str, Any]:
    source_root = source_root.resolve()
    manifest_path = source_root / "installer" / "manifest.json"
    manifest = _load_json_object(manifest_path, "安装清单")
    if manifest.get("schema_version") != 1 or manifest.get("managed_by") != MANAGED_BY:
        raise InstallationError("安装清单身份校验失败。")
    version = manifest.get("version")
    files = manifest.get("files")
    if not isinstance(version, str) or not version.strip() or not isinstance(files, dict):
        raise InstallationError("安装清单版本或文件列表无效。")
    if set(files) != PACKAGE_FILES:
        raise InstallationError("安装清单中的文件范围不正确。")
    version_path = source_root / "VERSION"
    try:
        package_version = version_path.read_text(encoding="utf-8").strip()
    except OSError as error:
        raise InstallationError(f"无法读取版本文件：{version_path}") from error
    if package_version != version:
        raise InstallationError("VERSION 与安装清单版本不一致。")
    for relative_path, expected_hash in files.items():
        if not isinstance(expected_hash, str) or len(expected_hash) != 64:
            raise InstallationError(f"安装清单哈希无效：{relative_path}")
        source_path = source_root / relative_path
        if source_path.is_symlink() or not source_path.is_file():
            raise InstallationError(f"安装包缺少普通文件：{relative_path}")
        if _sha256(source_path) != expected_hash:
            raise InstallationError(f"安装包文件校验失败：{relative_path}")
    return manifest


def _manifest_bytes(manifest: dict[str, Any]) -> bytes:
    return (json.dumps(manifest, ensure_ascii=False, indent=2, sort_keys=True) + "\n").encode(
        "utf-8"
    )


def _desired_files(
    source_root: Path, layout: Layout, manifest: dict[str, Any]
) -> dict[Path, bytes]:
    return {
        layout.tool_dir / "生成图片.py": (source_root / "tool" / "生成图片.py").read_bytes(),
        layout.tool_dir / "VERSION": (source_root / "VERSION").read_bytes(),
        layout.state_path: _manifest_bytes(manifest),
        layout.skill_dir / "SKILL.md": (
            source_root / "skill-template" / SKILL_NAME / "SKILL.md"
        ).read_bytes(),
        layout.skill_dir / "agents" / "openai.yaml": (
            source_root / "skill-template" / SKILL_NAME / "agents" / "openai.yaml"
        ).read_bytes(),
        layout.tool_dir / KEY_SOURCE_FILE_NAME: KEY_SOURCE_DEDICATED.encode("utf-8"),
    }


def _ensure_directory_target(path: Path, label: str) -> None:
    if path.is_symlink() or (path.exists() and not path.is_dir()):
        raise InstallationError(f"{label}不是可管理的普通目录：{path}")


def _reject_symlink_components(base: Path, target: Path, label: str) -> None:
    try:
        relative_parts = target.relative_to(base).parts
    except ValueError as error:
        raise InstallationError(f"{label}不在 Codex Home 内：{target}") from error
    current = base
    for part in relative_parts:
        current /= part
        if current.is_symlink():
            raise InstallationError(f"{label}路径包含符号链接，停止操作：{current}")


def _check_managed_path_components(layout: Layout) -> None:
    for target, label in (
        (layout.tool_dir, "工具目录"),
        (layout.skill_dir, "Skill 目录"),
        (layout.skill_dir / "agents", "Skill 子目录"),
        (layout.agents_path, "AGENTS.md"),
        (layout.override_path, "AGENTS.override.md"),
    ):
        _reject_symlink_components(layout.codex_home, target, label)


def _read_installed_state(layout: Layout, manifest: dict[str, Any]) -> dict[str, Any] | None:
    _ensure_directory_target(layout.tool_dir, "工具目录")
    _ensure_directory_target(layout.skill_dir, "Skill 目录")
    if layout.state_path.is_symlink() or layout.state_path.exists():
        if layout.state_path.is_symlink() or not layout.state_path.is_file():
            raise InstallationError(f"安装状态文件不是普通文件：{layout.state_path}")
        state = _load_json_object(layout.state_path, "安装状态")
        if state.get("managed_by") != MANAGED_BY or state.get("schema_version") != 1:
            raise InstallationError(f"目标目录无法确认由本工具管理：{layout.tool_dir}")
        installed_version = state.get("version")
        if installed_version not in {manifest["version"], "0.2.0", "0.3.0", "0.4.0", "0.4.1", "0.5.0", "0.6.0", "0.7.0", "0.7.1", "0.7.2", "0.7.3", "0.7.4", "0.7.5", "0.7.6", "0.8.0"}:
            raise InstallationError(
                f"已安装版本为 {installed_version}，当前安装包为 {manifest['version']}；不支持直接升级。"
            )
        return state

    if layout.tool_dir.is_symlink() or layout.tool_dir.exists():
        unknown = sorted(
            path.name
            for path in layout.tool_dir.iterdir()
            if path.name not in {LEGACY_KEY_FILE_NAME, KEY_SOURCE_FILE_NAME}
        )
        if unknown:
            raise InstallationError(f"目标目录无法确认由本工具管理：{layout.tool_dir}")
    if layout.skill_dir.is_symlink():
        raise InstallationError(f"Skill 目录不是可管理的普通目录：{layout.skill_dir}")
    if layout.skill_dir.exists() and any(layout.skill_dir.iterdir()):
        raise InstallationError(f"目标目录无法确认由本工具管理：{layout.skill_dir}")
    return None


def _read_agents(layout: Layout, *, check_override: bool = True) -> str:
    if check_override and (layout.override_path.is_symlink() or layout.override_path.exists()):
        if layout.override_path.is_symlink() or not layout.override_path.is_file():
            raise InstallationError(f"AGENTS.override.md 不是普通文件：{layout.override_path}")
        try:
            override_is_nonempty = layout.override_path.stat().st_size > 0
        except OSError as error:
            raise InstallationError(f"无法检查 AGENTS.override.md：{layout.override_path}") from error
        if override_is_nonempty:
            raise InstallationError(
                f"发现非空 AGENTS.override.md，会覆盖同级 AGENTS.md：{layout.override_path}"
            )
    if layout.agents_path.is_symlink():
        raise InstallationError(f"AGENTS.md 不是普通文件：{layout.agents_path}")
    if not layout.agents_path.exists():
        return ""
    if not layout.agents_path.is_file():
        raise InstallationError(f"AGENTS.md 不是普通文件：{layout.agents_path}")
    try:
        content = layout.agents_path.read_text(encoding="utf-8")
    except (OSError, UnicodeDecodeError) as error:
        raise InstallationError(f"无法读取 AGENTS.md：{layout.agents_path}") from error
    _marker_span(content)
    return content


def _marker_span(content: str) -> tuple[int, int] | None:
    start_count = content.count(RULE_START)
    end_count = content.count(RULE_END)
    if start_count == 0 and end_count == 0:
        return None
    if start_count != 1 or end_count != 1:
        raise InstallationError("AGENTS.md 中的中转站生图标记缺失、重复或不完整。")
    start = content.index(RULE_START)
    end = content.index(RULE_END)
    if end < start:
        raise InstallationError("AGENTS.md 中的中转站生图标记顺序错误。")
    return start, end + len(RULE_END)


def _install_rule(content: str) -> str:
    span = _marker_span(content)
    if span is not None:
        return content[: span[0]] + RULE_BLOCK + content[span[1] :]
    if not content:
        return RULE_BLOCK + "\n"
    separator = "\n\n" if content.endswith("\n") else "\n\n"
    return content.rstrip("\n") + separator + RULE_BLOCK + "\n"


def _remove_rule(content: str) -> str:
    span = _marker_span(content)
    if span is None:
        raise InstallationError("AGENTS.md 中缺少本工具规则块，停止卸载。")
    return content[: span[0]] + content[span[1] :]


def _same_file_bytes(path: Path, expected: bytes) -> bool:
    if path.is_symlink() or (path.exists() and not path.is_file()):
        raise InstallationError(f"托管文件路径发生冲突：{path}")
    try:
        return path.is_file() and path.read_bytes() == expected
    except OSError as error:
        raise InstallationError(f"无法检查托管文件：{path}") from error


def _atomic_write(path: Path, content: bytes) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    existing_mode: int | None = None
    if path.is_symlink():
        raise InstallationError(f"文件路径发生冲突：{path}")
    if path.exists() and os.name != "nt":
        existing_mode = stat.S_IMODE(path.stat().st_mode)
    temporary_path = path.parent / f".myautoapi-tmp-{os.getpid()}-{uuid.uuid4().hex}"
    descriptor: int | None = None
    try:
        descriptor = os.open(temporary_path, os.O_WRONLY | os.O_CREAT | os.O_EXCL, 0o666)
        with os.fdopen(descriptor, "wb") as handle:
            descriptor = None
            handle.write(content)
            handle.flush()
            os.fsync(handle.fileno())
        if existing_mode is not None:
            os.chmod(temporary_path, existing_mode)
        os.replace(temporary_path, path)
    except Exception:
        if descriptor is not None:
            os.close(descriptor)
        try:
            temporary_path.unlink()
        except FileNotFoundError:
            pass
        raise


class _FileTransaction:
    def __init__(self) -> None:
        self._temporary = tempfile.TemporaryDirectory(prefix="myautoapi-install-")
        self._backup_root = Path(self._temporary.name)
        self._staging_root = self._backup_root / "staged"
        self._staging_root.mkdir()
        self._journal: list[tuple[Path, Path | None]] = []
        self._finished = False

    def stage(self, name: str, content: bytes) -> Path:
        staged_path = self._staging_root / name
        staged_path.parent.mkdir(parents=True, exist_ok=True)
        staged_path.write_bytes(content)
        return staged_path

    def write(self, path: Path, content: bytes) -> None:
        backup: Path | None = None
        if path.is_symlink() or path.exists():
            if not path.is_file():
                raise InstallationError(f"文件路径发生冲突：{path}")
            backup = self._backup_root / f"backup-{len(self._journal)}"
            shutil.copy2(path, backup)
        self._journal.append((path, backup))
        _atomic_write(path, content)

    def remove(self, path: Path) -> None:
        if not path.exists() and not path.is_symlink():
            return
        if path.is_symlink() or not path.is_file():
            raise InstallationError(f"文件路径发生冲突：{path}")
        backup = self._backup_root / f"backup-{len(self._journal)}"
        shutil.copy2(path, backup)
        self._journal.append((path, backup))
        path.unlink()

    def rollback(self) -> list[Path]:
        if self._finished:
            return []
        failures: list[Path] = []
        for path, backup in reversed(self._journal):
            try:
                if backup is None:
                    path.unlink(missing_ok=True)
                else:
                    path.parent.mkdir(parents=True, exist_ok=True)
                    shutil.copy2(backup, path)
            except OSError:
                failures.append(path)
        self._finished = True
        self._temporary.cleanup()
        return failures

    def commit(self) -> None:
        if self._finished:
            return
        self._finished = True
        self._temporary.cleanup()


def _backup_path(agents_path: Path) -> Path:
    timestamp = datetime.now().astimezone().strftime("%Y%m%d-%H%M%S-%f")
    return agents_path.with_name(f"AGENTS.md.myautoapi-backup-{timestamp}")


def _call_hook(failure_hook: FailureHook | None, stage: str) -> None:
    if failure_hook is not None:
        failure_hook(stage)


def _cleanup_empty_directories(layout: Layout) -> None:
    candidates = [
        layout.skill_dir / "agents",
        layout.skill_dir,
        layout.codex_home / "skills",
        layout.tool_dir,
        layout.codex_home / "tools",
    ]
    for path in candidates:
        try:
            path.rmdir()
        except OSError:
            pass


def _nonempty_regular_file(path: Path) -> bool:
    try:
        return not path.is_symlink() and path.is_file() and path.stat().st_size > 0
    except OSError:
        return False


def _read_line(input_stream: TextIO, output: TextIO, prompt: str) -> str:
    print(prompt, end="", file=output, flush=True)
    line = input_stream.readline()
    if line == "":
        raise InstallationError("未收到输入，安装已停止。")
    return line.strip()


def _confirm_default_yes(input_stream: TextIO, output: TextIO, prompt: str) -> bool:
    while True:
        print(prompt, file=output)
        choice = _read_line(input_stream, output, "")
        if choice.lower() in {"", "y"}:
            return True
        if choice.lower() == "n":
            return False
        print("请输入 Y 或 N，直接回车表示同意。", file=output)


def _prompt_dedicated_key(input_stream: TextIO, output: TextIO) -> str:
    print("现在需要填写‘专门画图分组’的 Key。", file=output)
    print("【请仔细核对】", file=output)
    print("请粘贴‘专门画图分组’的 Key。", file=output)
    print("不要把‘编程分组 Key’和‘专门画图分组 Key’复制反。", file=output)
    print("安装器无法分辨 Key 是否复制错，必须确保自己复制正确。", file=output)
    while True:
        key = _read_line(input_stream, output, "请输入专门画图分组 Key：")
        if key:
            return key
        print("Key 不能为空，请重新输入。", file=output)


def _load_programming_key(layout: Layout) -> str:
    auth_path = layout.codex_home / "auth.json"
    if auth_path.is_symlink() or not auth_path.is_file():
        raise InstallationError(f"无法检查前面填写的编程分组 Key：{auth_path}")
    auth = _load_json_object(auth_path, "Codex auth.json")
    key = auth.get("OPENAI_API_KEY")
    if not isinstance(key, str) or not key.strip():
        raise InstallationError("Codex auth.json 中缺少可用的 OPENAI_API_KEY。")
    return key.strip()


def _capture_secret(path: Path) -> tuple[bool, bytes, int | None]:
    if path.is_symlink():
        raise InstallationError(f"文件路径发生冲突：{path}")
    if not path.exists():
        return False, b"", None
    if not path.is_file():
        raise InstallationError(f"文件路径发生冲突：{path}")
    try:
        mode = stat.S_IMODE(path.stat().st_mode) if os.name != "nt" else None
        return True, path.read_bytes(), mode
    except OSError as error:
        raise InstallationError(f"无法读取原有专门画图分组 Key：{path}") from error


def _restore_secret(path: Path, backup: tuple[bool, bytes, int | None]) -> None:
    existed, content, mode = backup
    if not existed:
        path.unlink(missing_ok=True)
        return
    _atomic_write(path, content)
    if mode is not None:
        os.chmod(path, mode)


def install(
    *,
    source_root: Path | str,
    environ: Mapping[str, str] | None = None,
    home: Path | str | None = None,
    output: TextIO | None = None,
    input_stream: TextIO | None = None,
    integrated: bool = False,
    compare_programming_key: bool = False,
    failure_hook: FailureHook | None = None,
) -> InstallResult:
    destination_output = sys.stdout if output is None else output
    source_input = sys.stdin if input_stream is None else input_stream
    print("将安装图片工具、‘中转站生图’ Skill 和全局默认生图规则。", file=destination_output)
    print("本次需要填写‘专门画图分组’的 Key；工具运行时只读取工具目录中的 Key。", file=destination_output)

    root = Path(source_root).resolve()
    manifest = validate_package(root)
    layout = build_layout(resolve_codex_home(environ, home))
    _check_managed_path_components(layout)
    state = _read_installed_state(layout, manifest)
    agents_content = _read_agents(layout)
    desired_agents = _install_rule(agents_content)
    desired = _desired_files(root, layout, manifest)

    changed_files = [path for path, content in desired.items() if not _same_file_bytes(path, content)]
    agents_changed = desired_agents != agents_content
    key_path = layout.tool_dir / LEGACY_KEY_FILE_NAME

    if integrated and not _confirm_default_yes(
        source_input,
        destination_output,
        "是否同时安装 MyAutoAPI 生图工具？\n直接回车安装；输入 N 跳过并退出生图安装。\n[Y/n]",
    ):
        print("已跳过生图工具安装。", file=destination_output)
        print(
            "当前 Codex 基础功能可以正常使用；以后需要生图时，再运行生图工具安装程序即可。",
            file=destination_output,
        )
        return InstallResult("skipped", layout.codex_home, layout.tool_dir, layout.skill_dir, layout.agents_path)

    pending_key = _prompt_dedicated_key(source_input, destination_output)
    if compare_programming_key and pending_key == _load_programming_key(layout):
        if not _confirm_default_yes(
            source_input,
            destination_output,
            "检测到‘专门画图分组 Key’和前面填写的‘编程分组 Key’完全相同。\n"
            "这可能是重复复制，请检查两个 Key 是否复制反了。\n"
            "安装器无法判断 Key 是否正确，必须由你确认。\n\n"
            "直接回车继续使用当前 Key；输入 N 退出生图安装。\n[Y/n]",
        ):
            print("已退出生图工具安装。", file=destination_output)
            print(
                "当前 Codex 基础功能可以正常使用；以后需要生图时，再重新运行生图工具安装程序即可。",
                file=destination_output,
            )
            return InstallResult("skipped", layout.codex_home, layout.tool_dir, layout.skill_dir, layout.agents_path)

    transaction = _FileTransaction()
    key_backup = _capture_secret(key_path)
    key_write_started = False
    try:
        staged = {
            path: transaction.stage(str(index), content)
            for index, (path, content) in enumerate(desired.items())
        }
        for path in changed_files:
            transaction.write(path, staged[path].read_bytes())
        if agents_changed:
            if layout.agents_path.exists():
                transaction.write(_backup_path(layout.agents_path), agents_content.encode("utf-8"))
            transaction.write(layout.agents_path, desired_agents.encode("utf-8"))
        key_write_started = True
        _atomic_write(key_path, pending_key.encode("utf-8"))
        _call_hook(failure_hook, "managed-files-written")
        for path, content in desired.items():
            if not _same_file_bytes(path, content):
                raise InstallationError(f"安装后校验失败：{path}")
        if layout.agents_path.read_text(encoding="utf-8") != desired_agents:
            raise InstallationError(f"全局规则写入后校验失败：{layout.agents_path}")
        if not _nonempty_regular_file(key_path):
            raise InstallationError(f"专门画图分组 Key 保存后校验失败：{key_path}")
        _call_hook(failure_hook, "validated")
    except Exception as error:
        rollback_failures = transaction.rollback()
        if key_write_started and key_backup is not None:
            try:
                _restore_secret(key_path, key_backup)
            except OSError:
                rollback_failures.append(key_path)
        _cleanup_empty_directories(layout)
        if rollback_failures:
            failed_paths = "、".join(str(path) for path in rollback_failures)
            raise InstallationError(f"{error}；回滚不完整，请检查：{failed_paths}") from error
        if isinstance(error, InstallationError):
            raise
        raise InstallationError(f"本地文件操作失败：{error}") from error
    transaction.commit()

    if state is None:
        status = "installed"
    elif state.get("version") != manifest["version"]:
        status = "upgraded"
    elif changed_files or agents_changed:
        status = "repaired"
    else:
        status = "configured"
    print("专门画图分组 Key 已保存。", file=destination_output)
    print(f"安装完成：V{manifest['version']}", file=destination_output)
    print(f"工具：{layout.tool_dir}", file=destination_output)
    print(f"Skill：{layout.skill_dir}", file=destination_output)
    print(f"全局规则：{layout.agents_path}", file=destination_output)
    print("请重新打开 Codex 或开始一个新任务，然后直接说：画一张白底红苹果。", file=destination_output)
    return InstallResult(
        status,
        layout.codex_home,
        layout.tool_dir,
        layout.skill_dir,
        layout.agents_path,
    )


def _assert_known_uninstall_files(layout: Layout) -> None:
    allowed_tool_files = {
        "生成图片.py",
        "VERSION",
        STATE_FILE_NAME,
        LEGACY_KEY_FILE_NAME,
        KEY_SOURCE_FILE_NAME,
    }
    if layout.tool_dir.exists():
        for path in layout.tool_dir.iterdir():
            if path.is_symlink() or not path.is_file() or path.name not in allowed_tool_files:
                raise InstallationError(f"工具目录包含未知文件，停止卸载：{path}")
    allowed_skill_files = {Path("SKILL.md"), Path("agents/openai.yaml")}
    if layout.skill_dir.exists():
        for path in layout.skill_dir.rglob("*"):
            if path.is_symlink():
                raise InstallationError(f"Skill 目录包含未知文件，停止卸载：{path}")
            if path.is_file() and path.relative_to(layout.skill_dir) not in allowed_skill_files:
                raise InstallationError(f"Skill 目录包含未知文件，停止卸载：{path}")
            if path.is_dir() and path.relative_to(layout.skill_dir) != Path("agents"):
                raise InstallationError(f"Skill 目录包含未知目录，停止卸载：{path}")


def uninstall(
    *,
    source_root: Path | str,
    environ: Mapping[str, str] | None = None,
    home: Path | str | None = None,
    output: TextIO | None = None,
    failure_hook: FailureHook | None = None,
) -> InstallResult:
    destination_output = sys.stdout if output is None else output
    print("将卸载图片工具、‘中转站生图’ Skill 和全局默认生图规则。", file=destination_output)
    print("卸载不会读取、修改或删除 Codex auth.json。", file=destination_output)

    root = Path(source_root).resolve()
    manifest = validate_package(root)
    layout = build_layout(resolve_codex_home(environ, home))
    _check_managed_path_components(layout)
    state = _read_installed_state(layout, manifest)
    if state is None:
        agents_content = ""
        if layout.agents_path.exists():
            agents_content = _read_agents(layout, check_override=False)
            if _marker_span(agents_content) is not None:
                raise InstallationError("发现规则块但缺少安装状态，无法确认卸载范围。")
        status = "already-uninstalled"
        print("未发现可确认的图片工具安装，无需卸载。", file=destination_output)
        return InstallResult(
            status,
            layout.codex_home,
            layout.tool_dir,
            layout.skill_dir,
            layout.agents_path,
        )

    _assert_known_uninstall_files(layout)
    agents_content = _read_agents(layout, check_override=False)
    desired_agents = _remove_rule(agents_content)
    transaction = _FileTransaction()
    try:
        if layout.agents_path.exists():
            transaction.write(_backup_path(layout.agents_path), agents_content.encode("utf-8"))
            transaction.write(layout.agents_path, desired_agents.encode("utf-8"))
        _call_hook(failure_hook, "rule-removed")
        for path in (
            layout.skill_dir / "agents" / "openai.yaml",
            layout.skill_dir / "SKILL.md",
            layout.tool_dir / "生成图片.py",
            layout.tool_dir / "VERSION",
            layout.tool_dir / KEY_SOURCE_FILE_NAME,
            layout.state_path,
        ):
            transaction.remove(path)
        _call_hook(failure_hook, "managed-files-removed")
    except Exception as error:
        rollback_failures = transaction.rollback()
        if rollback_failures:
            failed_paths = "、".join(str(path) for path in rollback_failures)
            raise InstallationError(f"{error}；回滚不完整，请检查：{failed_paths}") from error
        if isinstance(error, InstallationError):
            raise
        raise InstallationError(f"本地文件操作失败：{error}") from error
    transaction.commit()
    _cleanup_empty_directories(layout)

    status = "uninstalled"
    print("卸载完成。", file=destination_output)
    return InstallResult(
        status,
        layout.codex_home,
        layout.tool_dir,
        layout.skill_dir,
        layout.agents_path,
    )


def create_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="安装或卸载 MyAutoAPI 图片工具 V0.8.1。")
    subparsers = parser.add_subparsers(dest="command", required=True)
    install_parser = subparsers.add_parser("install", help="安装或修复 V0.8.1。")
    install_parser.add_argument("--integrated", action="store_true", help=argparse.SUPPRESS)
    install_parser.add_argument(
        "--source-root",
        default=str(Path(__file__).resolve().parents[1]),
        help=argparse.SUPPRESS,
    )
    uninstall_parser = subparsers.add_parser("uninstall", help="卸载 V0.8.1。")
    uninstall_parser.add_argument(
        "--source-root",
        default=str(Path(__file__).resolve().parents[1]),
        help=argparse.SUPPRESS,
    )
    return parser


def main(
    argv: list[str] | None = None,
    *,
    environ: Mapping[str, str] | None = None,
) -> int:
    arguments = create_parser().parse_args(argv)
    action = "安装" if arguments.command == "install" else "卸载"
    try:
        if arguments.command == "install":
            install(
                source_root=arguments.source_root,
                environ=environ,
                integrated=arguments.integrated,
                compare_programming_key=arguments.integrated,
            )
        else:
            uninstall(
                source_root=arguments.source_root,
                environ=environ,
            )
    except InstallationError as error:
        print(f"{action}失败：{error}", file=sys.stderr)
        print("本次操作未完成；写入前已停止或已恢复本次修改。", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
