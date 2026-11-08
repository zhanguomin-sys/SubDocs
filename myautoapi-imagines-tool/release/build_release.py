#!/usr/bin/env python3
"""Build the retained legacy Go implementation only when explicitly requested."""

from __future__ import annotations

import argparse
import hashlib
import os
import re
import shutil
import subprocess
from pathlib import Path


PROJECT_ROOT = Path(__file__).resolve().parents[1]
RELEASE_REVISION = "v0.9.2"
ARTIFACT_PREFIX = f"myautoapi-imagines-tool-{RELEASE_REVISION}"
TARGETS = (
    ("windows", "amd64"),
    ("darwin", "arm64"),
    ("darwin", "amd64"),
    ("linux", "amd64"),
)


def artifact_name(goos: str, goarch: str) -> str:
    suffix = ".exe" if goos == "windows" else ""
    return f"{ARTIFACT_PREFIX}-{goos}-{goarch}{suffix}"


def current_artifact_name(goos: str, goarch: str) -> str:
    suffix = ".exe" if goos == "windows" else ""
    return f"myautoapi-imagines-tool-{goos}-{goarch}{suffix}"


def sha256(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for chunk in iter(lambda: handle.read(1024 * 1024), b""):
            digest.update(chunk)
    return digest.hexdigest()


def update_release_script_hashes(hashes: dict[str, str]) -> None:
    for relative_path in ("release/install.sh",):
        path = PROJECT_ROOT / relative_path
        content = path.read_text(encoding="utf-8")
        for artifact, digest in hashes.items():
            if "-windows-" in artifact:
                continue
            pattern = rf'(artifact="{re.escape(artifact)}"\n\s*expected_sha256=")[0-9a-f]{{64}}(")'
            content, count = re.subn(pattern, rf"\g<1>{digest}\2", content)
            if count != 1:
                raise SystemExit(f"发布失败：无法更新 {relative_path} 中的 {artifact} 哈希。")
        path.write_text(content, encoding="utf-8")

    for relative_path in ("release/install.ps1",):
        path = PROJECT_ROOT / relative_path
        content = path.read_text(encoding="utf-8")
        for artifact, digest in hashes.items():
            if "-windows-" not in artifact:
                continue
            pattern = rf'(\$Artifact = "{re.escape(artifact)}"\n\s*\$ExpectedSha256 = ")[0-9a-f]{{64}}(")'
            content, count = re.subn(pattern, rf"\g<1>{digest}\2", content)
            if count != 1:
                raise SystemExit(f"发布失败：无法更新 {relative_path} 中的 {artifact} 哈希。")
        path.write_text(content, encoding="utf-8")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--legacy-source",
        action="store_true",
        help="explicitly build the historical Go source; not the current release",
    )
    arguments = parser.parse_args()
    if not arguments.legacy_source:
        parser.error(
            "当前发布制品来自固定上游原生程序，请使用 release/port_upstream.py <上游仓库目录>。"
            "保留的 Go 源码不是当前发布实现；仅历史复现可显式传入 --legacy-source。"
        )
    print(
        "警告：正在构建历史 Go 实现，输出不得替换当前正式发布制品、安装脚本或 latest.json。"
    )
    if (PROJECT_ROOT / "VERSION").read_text(encoding="utf-8").strip() != "0.9.2":
        raise SystemExit("发布失败：V0.9.2 必须使用安装状态版本 0.9.2。")

    subprocess.run(["go", "test", "./..."], cwd=PROJECT_ROOT, check=True)
    output_dir = PROJECT_ROOT / "dist" / ("legacy-" + RELEASE_REVISION)
    output_dir.mkdir(parents=True, exist_ok=True)
    for stale_artifact in output_dir.glob(f"{ARTIFACT_PREFIX}-*"):
        if stale_artifact.is_file():
            stale_artifact.unlink()
    artifacts: list[Path] = []
    for goos, goarch in TARGETS:
        output_path = output_dir / artifact_name(goos, goarch)
        environment = os.environ.copy()
        environment.update({"GOOS": goos, "GOARCH": goarch, "CGO_ENABLED": "0"})
        subprocess.run(
            [
                "go",
                "build",
                "-trimpath",
                "-buildvcs=false",
                "-ldflags=-s -w",
                "-o",
                str(output_path),
                ".",
            ],
            cwd=PROJECT_ROOT,
            env=environment,
            check=True,
        )
        artifacts.append(output_path)

    hashes = {path.name: sha256(path) for path in artifacts}
    checksum_path = output_dir / "SHA256SUMS"
    checksum_path.write_text(
        "".join(f"{hashes[path.name]}  {path.name}\n" for path in artifacts),
        encoding="ascii",
    )
    current_dir = output_dir / "current"
    current_dir.mkdir(parents=True, exist_ok=True)
    for stale_artifact in current_dir.iterdir():
        if stale_artifact.is_file():
            stale_artifact.unlink()
    current_artifacts: list[Path] = []
    current_hashes: dict[str, str] = {}
    for (goos, goarch), artifact in zip(TARGETS, artifacts):
        current_path = current_dir / current_artifact_name(goos, goarch)
        shutil.copyfile(artifact, current_path)
        current_artifacts.append(current_path)
        current_hashes[current_path.name] = hashes[artifact.name]
    (current_dir / "SHA256SUMS").write_text(
        "".join(f"{current_hashes[path.name]}  {path.name}\n" for path in current_artifacts),
        encoding="ascii",
    )
    # Historical builds must not rewrite the current upstream-based installers.
    print(output_dir)
    print(checksum_path.read_text(encoding="ascii"), end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
