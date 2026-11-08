#!/usr/bin/env python3
"""Rebrand the pinned upstream V0.9.2 programs without changing their logic."""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import struct
import tempfile

ROOT = Path(__file__).resolve().parents[2]
UPSTREAM_COMMIT = 'c28a91174db4f065ff4c4121e348f20afebda85c'
VERSION = '0.9.2'
ARCHIVE = 'v0.9.2-upstream-c28a911'
UPSTREAM_HASHES = {
    'windows/amd64': '11947f21ae3a591bf46f092f322c7ad7cc97758edf7cdbaf344f9a423ca8f282',
    'darwin/arm64': '13d1941c91c84e668aee11bb808910607897a02dcb282d8e8ca906a2172677b4',
    'darwin/amd64': 'f993f4cd429cebd811429cdacb129de6ce1f283d3e844f0b5de7cc4c16404428',
    'linux/amd64': 'e629977b7490d26063f7f6df23cfd5a3421117414ad080e8c328621880c9014c',
}
REPLACEMENTS = ((b'usegoodai', b'myautoapi'), (b'UseGoodAI', b'MyAutoAPI'), (b'USEGOODAI', b'MYAUTOAPI'))

def digest(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()

def refresh_macho_signature(data: bytearray) -> int:
    """Refresh Go's existing ad-hoc SHA-256 CodeDirectory, keeping its layout."""
    if data[:4] != b'\xcf\xfa\xed\xfe':
        return 0
    commands = struct.unpack_from('<I', data, 16)[0]
    offset = 32
    for _ in range(commands):
        command, length = struct.unpack_from('<II', data, offset)
        if command == 0x1d:  # LC_CODE_SIGNATURE
            start, size = struct.unpack_from('<II', data, offset + 8)
            magic, total, count = struct.unpack_from('>III', data, start)
            if magic != 0xfade0cc0 or total > size or start + size > len(data) or count != 1:
                raise ValueError('Unsupported Mach-O signature container')
            slot, relative = struct.unpack_from('>II', data, start + 12)
            directory = start + relative
            cd_magic, cd_length, version, flags, hashes, identifier, special, pages, limit = struct.unpack_from('>9I', data, directory)
            hash_size, hash_type, platform, page_exponent = struct.unpack_from('4B', data, directory + 36)
            if slot != 0 or cd_magic != 0xfade0c02 or special != 0 or hash_size != 32 or hash_type != 2 or not flags & 2:
                raise ValueError('Only an existing ad-hoc SHA-256 signature is supported')
            page_size = 1 << page_exponent
            if limit != start or pages != (limit + page_size - 1) // page_size or hashes + pages * 32 > cd_length:
                raise ValueError('Invalid CodeDirectory bounds')
            for page in range(pages):
                begin = page * page_size
                actual = hashlib.sha256(data[begin:min(begin + page_size, limit)]).digest()
                target = directory + hashes + page * 32
                data[target:target + 32] = actual
            return pages
        if length < 8:
            raise ValueError('Invalid Mach-O command')
        offset += length
    return 0


def transform_platform(source: Path, platform: str, expected: str) -> dict[str, object]:
    """Read, transform, and validate one upstream artifact without writing it."""
    suffix = platform.replace('/', '-') + ('.exe' if platform.startswith('windows') else '')
    upstream_name = 'usegoodai-imagines-tool-v0.9.2-' + suffix
    original = (source / upstream_name).read_bytes()
    if digest(original) != expected:
        raise ValueError('Upstream SHA-256 mismatch: ' + upstream_name)
    updated = original
    counts = {}
    for before, after in REPLACEMENTS:
        assert len(before) == len(after)
        counts[before.decode()] = updated.count(before)
        updated = updated.replace(before, after)
    result = bytearray(updated)
    signed_pages = refresh_macho_signature(result)
    if len(result) != len(original):
        raise ValueError('Branded artifact size changed: ' + upstream_name)
    if any(before.lower() in result.lower() for before, _ in REPLACEMENTS):
        raise ValueError('Upstream brand remains in: ' + upstream_name)
    for required in (b'https://api.myautoapi.com', b'https://docs.myautoapi.com/install/myautoapi-imagines-tool/latest.json'):
        if required not in result:
            raise ValueError('Missing branded endpoint in ' + upstream_name + ': ' + required.decode())
    artifact = 'myautoapi-imagines-tool-v0.9.2-' + suffix
    current_name = 'myautoapi-imagines-tool-' + suffix
    branded = bytes(result)
    return {
        'platform': platform,
        'artifact': artifact,
        'current_name': current_name,
        'bytes': branded,
        'sha256': digest(branded),
        'size': len(branded),
        'replacements': counts,
        'signed_pages': signed_pages,
    }


def prepare_installer(path: Path, hashes: dict[str, str], extension: str) -> str:
    """Prepare an installer copy and validate every expected hash replacement."""
    text = path.read_text(encoding='utf-8-sig')
    for name, sha in hashes.items():
        if (extension == 'ps1') != name.endswith('.exe'):
            continue
        pattern = (r'(\$Artifact = "' + re.escape(name) + r'"\n\$ExpectedSha256 = ")[0-9a-f]{64}(")') if extension == 'ps1' else (r'(artifact="' + re.escape(name) + r'"\n\s*expected_sha256=")[0-9a-f]{64}(")')
        text, count = re.subn(pattern, lambda match: match[1] + sha + match[2], text)
        if count != 1:
            raise ValueError('Cannot update installer hash: ' + name)
    # Explicit bootstrap/Key configuration always accepts the entered Key; update preserves it.
    if extension == 'ps1':
        text = text.replace('& $BinaryPath install @args', '& $BinaryPath install --key-stdin @args')
        if '--key-stdin @args' not in text:
            raise ValueError('PowerShell installer does not pass --key-stdin')
    else:
        text = text.replace('install "$@" </dev/tty', 'install --key-stdin "$@" </dev/tty')
        if 'install --key-stdin "$@" </dev/tty' not in text:
            raise ValueError('Shell installer does not pass --key-stdin')
    return text


def write_staged_files(files: dict[Path, tuple[bytes, int]]) -> None:
    """Stage all files, then replace destinations one by one after validation."""
    if not files:
        return
    parent = next(iter(files)).parent
    while not parent.exists():
        parent = parent.parent
    with tempfile.TemporaryDirectory(prefix='.myautoapi-port-', dir=str(parent)) as temporary:
        stage_root = Path(temporary)
        staged: list[tuple[Path, Path, int]] = []
        for index, (destination, (content, mode)) in enumerate(files.items()):
            relative = Path(f'{index:04d}-{destination.name}')
            stage_path = stage_root / relative
            stage_path.write_bytes(content)
            stage_path.chmod(mode)
            staged.append((stage_path, destination, mode))
        for stage_path, destination, mode in staged:
            destination.parent.mkdir(parents=True, exist_ok=True)
            os.replace(stage_path, destination)
            if mode & 0o111:
                destination.chmod(mode)

def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('upstream', type=Path, help='Pinned youdaowudao/subdocs checkout')
    args = parser.parse_args()
    source = args.upstream / 'docs/public/install/usegoodai-imagines-tool/releases/v0.9.2'
    public = ROOT / 'docs/public/install/myautoapi-imagines-tool'
    release = public / 'releases' / ARCHIVE
    current = public / 'current'
    manifest = {'version': VERSION, 'release': 'V' + VERSION, 'files': {}}
    record = {'upstream_repository': 'https://github.com/youdaowudao/subdocs', 'upstream_commit': UPSTREAM_COMMIT, 'version': VERSION, 'archive': ARCHIVE, 'method': 'equal-length brand replacement; refresh existing Apple Silicon ad-hoc SHA-256 page hashes', 'files': {}}
    release_sums, current_sums = [], []
    hashes = {}
    transformed: list[dict[str, object]] = []
    for platform, expected in UPSTREAM_HASHES.items():
        item = transform_platform(source, platform, expected)
        transformed.append(item)
        artifact = item['artifact']
        current_name = item['current_name']
        sha = item['sha256']
        hashes[current_name] = sha
        release_sums.append(f'{sha}  {artifact}\n')
        current_sums.append(f'{sha}  {current_name}\n')
        manifest['files'][platform] = {'path': 'current/' + current_name, 'sha256': sha, 'size': item['size']}
        record['files'][platform] = {'upstream_sha256': expected, 'sha256': sha, 'size': item['size'], 'replacements': item['replacements'], 'signed_pages': item['signed_pages']}
    installers = {}
    for extension in ('ps1', 'sh'):
        installers[extension] = prepare_installer(ROOT / ('myautoapi-imagines-tool/release/install.' + extension), hashes, extension)
    manifest_bytes = (json.dumps(manifest, ensure_ascii=False, indent=2) + '\n').encode('utf-8')
    record_bytes = (json.dumps(record, ensure_ascii=False, indent=2) + '\n').encode('utf-8')
    files: dict[Path, tuple[bytes, int]] = {
        public / 'latest.json': (manifest_bytes, 0o644),
        ROOT / 'myautoapi-imagines-tool/release/upstream-port.json': (record_bytes, 0o644),
        ROOT / 'myautoapi-imagines-tool/release/install.ps1': (installers['ps1'].encode('utf-8'), 0o644),
        ROOT / 'myautoapi-imagines-tool/release/install.sh': (installers['sh'].encode('utf-8'), 0o644),
        public / 'install.ps1': (installers['ps1'].encode('utf-8'), 0o644),
        public / 'install.sh': (installers['sh'].encode('utf-8'), 0o644),
        release / 'SHA256SUMS': (''.join(release_sums).encode('ascii'), 0o644),
        current / 'SHA256SUMS': (''.join(current_sums).encode('ascii'), 0o644),
    }
    for item in transformed:
        mode = 0o644 if str(item['artifact']).endswith('.exe') else 0o755
        files[release / str(item['artifact'])] = (item['bytes'], mode)
        files[current / str(item['current_name'])] = (item['bytes'], mode)
    write_staged_files(files)
    print(json.dumps(record, ensure_ascii=False, indent=2))

if __name__ == '__main__':
    main()
