#!/usr/bin/env python3
"""Build the portable MyAutoAPI Windows x64 assistant and its release manifest."""

import hashlib
import json
import os
from pathlib import Path
import re
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parent
PUBLIC = ROOT.parents[1] / "docs" / "public" / "download"


def main():
    source = (ROOT / "config.go").read_text(encoding="utf-8")
    version = re.search(r'\bversion\s*=\s*"(\d+\.\d+\.\d+)"', source)[1]
    name = f"myautoapi-quick-connect-{version}-windows-x64.exe"
    release = PUBLIC / "quick-connect" / name
    env = dict(os.environ, GOOS="windows", GOARCH="amd64", CGO_ENABLED="0")
    with tempfile.TemporaryDirectory(prefix="myautoapi-assistant-") as temp_dir:
        binary = Path(temp_dir) / name
        subprocess.run(
            ["go", "build", "-mod=readonly", "-trimpath", "-buildvcs=false",
             "-tags", "desktop,production,wv2runtime.browser",
             "-ldflags", "-s -w -H windowsgui", "-o", str(binary), "."],
            env=env, cwd=ROOT, check=True,
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
    (PUBLIC / "myautoapi-quick-connect-windows-x64.json").write_text(
        json.dumps(manifest, indent=2) + "\n", encoding="utf-8",
    )
    print(f"{release}: {len(payload)} bytes, SHA-256 {digest}")


if __name__ == "__main__":
    main()
