#!/usr/bin/env python3
"""Build the portable MyAutoAPI Windows x64 assistant and its release manifest."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import shutil
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parent
PUBLIC = ROOT.parents[1] / "docs" / "public"
INSTALL = PUBLIC / "install"
DOWNLOAD = PUBLIC / "download"


def release_bytes(root, item):
    path = (root / item["path"]).resolve()
    if not path.is_relative_to(root.resolve()):
        raise SystemExit("Release path escaped its package directory")
    data = path.read_bytes()
    if len(data) != item["size"] or hashlib.sha256(data).hexdigest() != item["sha256"]:
        raise SystemExit(f"Incorrect release checksum: {path}")
    return data


def prepare_bundle():
    """Package the same released EXE and installer used by the documentation."""
    image_root = INSTALL / "myautoapi-imagines-tool"
    image_release = json.loads((image_root / "latest.json").read_bytes())
    image = {"version": image_release["version"], **image_release["files"]["windows/amd64"]}
    image_bytes = release_bytes(image_root, image)
    if not image_bytes.startswith(b"MZ"):
        raise SystemExit("Expected a Windows image tool executable")

    models_root = INSTALL / "codex-models"
    models = json.loads((models_root / "latest.json").read_bytes())
    catalog = json.loads(release_bytes(models_root, models))
    installer_root = ROOT.parent / "codex-domestic-models"
    models_script = (installer_root / "install" / "codex-models.ps1").read_bytes().replace(b"\r\n", b"\n")
    web_script = (INSTALL / "codex-models.ps1").read_bytes().replace(b"\r\n", b"\n")
    if (models["version"] != (installer_root / "VERSION").read_text().strip() or
            not models_script.startswith(b"\xef\xbb\xbf") or models_script[3:] != web_script or
            b"-KeepClientsOpen:" not in models_script or b"$CatalogFile)" not in models_script):
        raise SystemExit("Rebuild and sync the repository's model installer before packaging")
    manifest = {
        "image": image, "models": models, "model_count": len(catalog["domestic"]["models"]),
        "models_script_sha256": hashlib.sha256(models_script).hexdigest(),
    }
    packages = ROOT / "packages"
    packages.mkdir(exist_ok=True)
    (packages / "image-tool.exe").write_bytes(image_bytes)
    (packages / "models.ps1").write_bytes(models_script)
    # The connection flow uses plain official model data from the same release.
    # Match the domestic installer's filter for the retired GPT-5.5 entry.
    official = dict(catalog["official"])
    official["models"] = [entry for entry in official["models"] if entry["slug"] != "gpt-5.5"]
    if not {"gpt-6.2-sol", "gpt-6.2-astra"} <= {entry["slug"] for entry in official["models"]}:
        raise SystemExit("The official catalog is missing the GPT-6.2 models")
    (packages / "official-models.json").write_text(
        json.dumps(official, ensure_ascii=False, separators=(",", ":")) + "\n", encoding="utf-8",
    )
    (packages / "release.json").write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--prepare-bundle", action="store_true", help="stage package files for Go tests")
    options = parser.parse_args()
    prepare_bundle()
    if options.prepare_bundle:
        return
    source = (ROOT / "config.go").read_text(encoding="utf-8")
    version = re.search(r'\bversion\s*=\s*"(\d+\.\d+\.\d+)"', source)[1]
    name = f"myautoapi-quick-connect-{version}-windows-x64.exe"
    release = DOWNLOAD / "quick-connect" / name
    host_env = os.environ.copy()
    host_env.pop("GOOS", None)
    host_env.pop("GOARCH", None)
    env = dict(host_env, GOOS="windows", GOARCH="amd64", CGO_ENABLED="0")
    with tempfile.TemporaryDirectory(prefix="myautoapi-assistant-") as temp_dir:
        work = Path(temp_dir)
        binary = work / name
        staged = work / "source"
        staged.mkdir()
        # The Windows linker reads .syso files from disk. Generate the icon in
        # a staged copy so release builds never leave linker objects in the checkout.
        for file in [ROOT / "go.mod", ROOT / "go.sum", *ROOT.glob("*.go")]:
            shutil.copyfile(file, staged / file.name)
        for directory in ("frontend", "internal", "packages", "locale"):
            shutil.copytree(ROOT / directory, staged / directory)
        subprocess.run(
            ["go", "run", "-mod=readonly", "./internal/winresources",
             "frontend/myautoapi.ico", "myautoapi_windows_amd64.syso"],
            env=host_env, cwd=staged, check=True,
        )
        subprocess.run(
            ["go", "build", "-mod=readonly", "-trimpath", "-buildvcs=false",
             "-tags", "desktop,production,wv2runtime.browser,quickconnect_bundle",
             "-ldflags", "-s -w -H windowsgui", "-o", str(binary), "."],
            env=env, cwd=staged, check=True,
        )
        payload = binary.read_bytes()
        if not payload.startswith(b"MZ") or len(payload) > 24 << 20:
            raise SystemExit("Unexpected executable or file too large for Pages")
    digest = hashlib.sha256(payload).hexdigest()
    if release.exists() and release.read_bytes() != payload:
        raise SystemExit("A different executable already exists for this version")
    release.parent.mkdir(parents=True, exist_ok=True)
    release.write_bytes(payload)
    manifest = {"version": version, "path": f"/download/quick-connect/{name}",
                "sha256": digest, "size": len(payload)}
    (DOWNLOAD / "myautoapi-quick-connect-windows-x64.json").write_text(
        json.dumps(manifest, indent=2) + "\n", encoding="utf-8",
    )
    print(f"{release}: {len(payload)} bytes, SHA-256 {digest}")


if __name__ == "__main__":
    main()
