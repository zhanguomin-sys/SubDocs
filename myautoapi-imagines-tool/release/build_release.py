#!/usr/bin/env python3
"""Build the self-contained V0.8.1 release artifacts."""

from __future__ import annotations

import hashlib
import os
import re
import subprocess
from pathlib import Path


PROJECT_ROOT = Path(__file__).resolve().parents[1]
RELEASE_REVISION = "v0.8.1"
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
    if (PROJECT_ROOT / "VERSION").read_text(encoding="utf-8").strip() != "0.8.1":
        raise SystemExit("发布失败：V0.8.1 必须使用安装状态版本 0.8.1。")

    subprocess.run(["go", "test", "./..."], cwd=PROJECT_ROOT, check=True)
    output_dir = PROJECT_ROOT / "dist" / RELEASE_REVISION
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
    update_release_script_hashes(hashes)
    print(output_dir)
    print(checksum_path.read_text(encoding="ascii"), end="")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
