"""Exercise port validation and staging with isolated, synthetic artifacts."""
from __future__ import annotations

import contextlib
import hashlib
import importlib.util
import io
import json
from pathlib import Path
import shutil
import stat
import tempfile
import unittest
from unittest.mock import patch

SCRIPT = Path(__file__).resolve().parents[1] / 'myautoapi-imagines-tool/release/port_upstream.py'
PLATFORMS = ('windows/amd64', 'darwin/arm64', 'darwin/amd64', 'linux/amd64')


def snapshot(root: Path) -> dict[str, tuple[object, ...]]:
    result = {}
    for path in sorted(root.rglob('*')):
        name = path.relative_to(root).as_posix()
        result[name] = ('file', path.read_bytes(), stat.S_IMODE(path.stat().st_mode)) if path.is_file() else ('directory',)
    return result


class PortPipelineTest(unittest.TestCase):
    def setUp(self) -> None:
        self.temporary = tempfile.TemporaryDirectory(prefix='myautoapi-port-pipeline-')
        self.addCleanup(self.temporary.cleanup)
        self.root = Path(self.temporary.name) / 'isolated-project'
        copied_script = self.root / 'myautoapi-imagines-tool/release/port_upstream.py'
        copied_script.parent.mkdir(parents=True)
        shutil.copyfile(SCRIPT, copied_script)
        spec = importlib.util.spec_from_file_location('isolated_port_pipeline', copied_script)
        self.port = importlib.util.module_from_spec(spec)
        spec.loader.exec_module(self.port)
        self.upstream = Path(self.temporary.name) / 'synthetic-upstream'
        self.source = self.upstream / 'docs/public/install/usegoodai-imagines-tool/releases/v0.9.2'
        self.source.mkdir(parents=True)
        hashes = {}
        self.current_names = {}
        for platform in PLATFORMS:
            suffix = platform.replace('/', '-') + ('.exe' if platform.startswith('windows') else '')
            payload = (
                'synthetic ' + platform + '\n'
                'https://api.usegoodai.com\n'
                'https://docs.usegoodai.com/install/usegoodai-imagines-tool/latest.json\n'
                'UseGoodAI USEGOODAI usegoodai\n'
            ).encode('ascii')
            (self.source / ('usegoodai-imagines-tool-v0.9.2-' + suffix)).write_bytes(payload)
            hashes[platform] = hashlib.sha256(payload).hexdigest()
            self.current_names[platform] = 'myautoapi-imagines-tool-' + suffix
        self.port.UPSTREAM_HASHES = hashes
        self.public = self.root / 'docs/public/install/myautoapi-imagines-tool'
        (self.public / 'current').mkdir(parents=True)
        for name in self.current_names.values():
            (self.public / 'current' / name).write_bytes(b'old artifact\n')
        (self.public / 'latest.json').write_text('{"previous":true}\n', encoding='utf-8')
        (self.public / 'current/SHA256SUMS').write_bytes(b'previous checksums\n')
        historical = self.public / 'releases/v0.9.2/SHA256SUMS'
        historical.parent.mkdir(parents=True)
        historical.write_bytes(b'unchanged historical archive\n')
        release = self.root / 'myautoapi-imagines-tool/release'
        win = self.current_names['windows/amd64']
        (release / 'install.ps1').write_text(
            '$Artifact = "' + win + '"\n$ExpectedSha256 = "' + '0' * 64 + '"\n$DedicatedImageKey = Read-Host "请输入专门画图分组 Key"\n$DedicatedImageKey | & $BinaryPath install --key-stdin @args\n',
            encoding='utf-8',
        )
        shell = ''.join(
            'artifact="' + name + '"\nexpected_sha256="' + '0' * 64 + '"\n'
            for platform, name in self.current_names.items() if not platform.startswith('windows')
        )
        (release / 'install.sh').write_text(shell + 'IFS= read -r dedicated_image_key </dev/tty\nprintf \'%s\\n\' "$dedicated_image_key" | "$binary_path" install --key-stdin "$@"\n', encoding='utf-8')

    def run_port(self) -> None:
        with patch('sys.argv', [str(SCRIPT), str(self.upstream)]), contextlib.redirect_stdout(io.StringIO()):
            self.port.main()

    def test_bad_last_platform_hash_does_not_modify_any_output(self) -> None:
        final_source = self.source / 'usegoodai-imagines-tool-v0.9.2-linux-amd64'
        final_source.write_bytes(final_source.read_bytes() + b'corrupted')
        before = snapshot(self.root)
        with self.assertRaisesRegex(ValueError, 'Upstream SHA-256 mismatch'):
            self.run_port()
        self.assertEqual(snapshot(self.root), before)

    def test_unmatched_late_installer_hash_does_not_modify_any_output(self) -> None:
        shell = self.root / 'myautoapi-imagines-tool/release/install.sh'
        shell.write_text(shell.read_text(encoding='utf-8').replace('myautoapi-imagines-tool-linux-amd64', 'unexpected-linux-artifact'), encoding='utf-8')
        before = snapshot(self.root)
        with self.assertRaisesRegex(ValueError, 'Cannot update installer hash'):
            self.run_port()
        self.assertEqual(snapshot(self.root), before)

    def test_silent_stdin_entry_does_not_modify_any_output(self) -> None:
        for extension, prompt in (('ps1', 'Read-Host'), ('sh', 'read -r dedicated_image_key </dev/tty')):
            installer = self.root / ('myautoapi-imagines-tool/release/install.' + extension)
            original = installer.read_text(encoding='utf-8')
            installer.write_text(original.replace(prompt, 'missing_prompt'), encoding='utf-8')
            before = snapshot(self.root)
            with self.subTest(extension=extension), self.assertRaisesRegex(ValueError, 'must prompt'):
                self.run_port()
            self.assertEqual(snapshot(self.root), before)
            installer.write_text(original, encoding='utf-8')

    def test_validated_staged_outputs_agree_and_keep_old_archive(self) -> None:
        historical = self.public / 'releases/v0.9.2/SHA256SUMS'
        original_archive = historical.read_bytes()
        self.run_port()
        self.assertEqual(historical.read_bytes(), original_archive)
        manifest = json.loads((self.public / 'latest.json').read_text(encoding='utf-8'))
        record = json.loads((self.root / 'myautoapi-imagines-tool/release/upstream-port.json').read_text(encoding='utf-8'))
        self.assertEqual(record['archive'], self.port.ARCHIVE)
        for platform, item in manifest['files'].items():
            current = (self.public / item['path']).read_bytes()
            self.assertEqual(hashlib.sha256(current).hexdigest(), item['sha256'])
            self.assertEqual(len(current), item['size'])
            suffix = self.current_names[platform].removeprefix('myautoapi-imagines-tool-')
            archived = self.public / 'releases' / self.port.ARCHIVE / ('myautoapi-imagines-tool-v0.9.2-' + suffix)
            self.assertEqual(archived.read_bytes(), current)
        for extension in ('ps1', 'sh'):
            internal = self.root / ('myautoapi-imagines-tool/release/install.' + extension)
            public = self.public / ('install.' + extension)
            self.assertEqual(internal.read_bytes(), public.read_bytes())
            self.assertIn(b'--key-stdin', public.read_bytes())
        self.assertEqual(list(self.root.rglob('.myautoapi-port-*')), [])


if __name__ == '__main__':
    unittest.main()
