#!/usr/bin/env python3
"""Exercise the ported Linux updater using only local HTTPS and synthetic keys."""
from __future__ import annotations

import hashlib
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import platform
import ssl
import subprocess
import tempfile
import threading
import unittest


ROOT = Path(__file__).resolve().parents[1]
PROGRAM = ROOT / 'docs/public/install/myautoapi-imagines-tool/current/myautoapi-imagines-tool-linux-amd64'
LATEST_URL = b'https://docs.myautoapi.com/install/myautoapi-imagines-tool/latest.json'


def sha256(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


@unittest.skipUnless(platform.system() == 'Linux' and platform.machine() in ('x86_64', 'amd64'), 'requires the Linux x64 release')
class NativeUpdateTest(unittest.TestCase):
    def test_same_version_requires_reinstall_and_checks_relative_current_payload(self) -> None:
        original = PROGRAM.read_bytes()
        with tempfile.TemporaryDirectory(prefix='myautoapi-update-local-') as temporary:
            root = Path(temporary)
            certificate, private_key = root / 'localhost.crt', root / 'localhost.key'
            subprocess.run([
                'openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes', '-days', '1',
                '-keyout', str(private_key), '-out', str(certificate), '-subj', '/CN=127.0.0.1',
                '-addext', 'subjectAltName=IP:127.0.0.1',
            ], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            state = {'requests': [], 'content': {}, 'unexpected': []}

            class Handler(BaseHTTPRequestHandler):
                def do_GET(self):
                    state['requests'].append(self.path)
                    content = state['content'].get(self.path)
                    if content is None:
                        state['unexpected'].append(self.path)
                        self.send_error(404)
                        return
                    self.send_response(200)
                    self.send_header('Content-Length', str(len(content)))
                    self.send_header('Content-Type', 'application/json' if self.path.endswith('.json') else 'application/octet-stream')
                    self.end_headers()
                    self.wfile.write(content)

                def do_CONNECT(self):
                    state['unexpected'].append('CONNECT ' + self.path)
                    self.send_error(403, 'external network disabled in this test')

                def log_message(self, *_):
                    pass

            server = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
            proxy = ThreadingHTTPServer(('127.0.0.1', 0), Handler)
            context = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
            context.load_cert_chain(certificate, private_key)
            server.socket = context.wrap_socket(server.socket, server_side=True)
            threads = [threading.Thread(target=item.serve_forever, daemon=True) for item in (server, proxy)]
            for thread in threads:
                thread.start()
            try:
                base = f'https://127.0.0.1:{server.server_port}/'
                padding = len(LATEST_URL) - len(base) - len('/latest.json')
                self.assertGreater(padding, 0)
                latest = (base + 'x' * padding + '/latest.json').encode()
                self.assertEqual(len(latest), len(LATEST_URL))
                self.assertEqual(original.count(LATEST_URL), 1)
                payload = original.replace(LATEST_URL, latest)
                # A harmless trailer creates an old V0.9.2 fixture with a different hash.
                old = payload + b'\nlocal-same-version-old-fixture\n'
                source = root / 'old-program'
                source.write_bytes(old)
                source.chmod(0o755)
                home = root / 'codex-home'
                environment = dict(os.environ, CODEX_HOME=str(home), HOME=str(root), SSL_CERT_FILE=str(certificate),
                    HTTPS_PROXY=f'http://127.0.0.1:{proxy.server_port}', HTTP_PROXY=f'http://127.0.0.1:{proxy.server_port}',
                    ALL_PROXY=f'http://127.0.0.1:{proxy.server_port}', NO_PROXY='127.0.0.1')
                # Keep proxy behavior deterministic even if the shell sets lowercase variants.
                for variable in ('https_proxy', 'http_proxy', 'all_proxy', 'no_proxy'):
                    environment.pop(variable, None)
                key = 'sk-' + 'local-synthetic-fixture-' + '0' * 48

                def run(program: Path, arguments: list[str], input: str = '', expected_code: int = 0):
                    completed = subprocess.run([str(program), *arguments], input=input, text=True, capture_output=True, env=environment, timeout=30)
                    self.assertEqual(completed.returncode, expected_code, completed.stderr or completed.stdout)
                    return completed.stdout + completed.stderr

                run(source, ['install', '--key-stdin'], key + '\n')
                installed = home / 'tools/myautoapi-imagines-tool/myautoapi-imagines-tool'
                key_path = installed.parent / 'api-key'
                self.assertEqual(run(installed, ['version']).strip(), '0.9.2')
                self.assertEqual(sha256(installed.read_bytes()), sha256(old))
                self.assertEqual(key_path.read_text().strip(), key)
                manifest_path = latest.decode().removeprefix(f'https://127.0.0.1:{server.server_port}')
                artifact_path = manifest_path.removesuffix('latest.json') + 'current/myautoapi-imagines-tool-linux-amd64'
                manifest = {'version': '0.9.2', 'release': 'V0.9.2', 'files': {'linux/amd64': {
                    'path': 'current/myautoapi-imagines-tool-linux-amd64', 'sha256': sha256(payload), 'size': len(payload),
                }}}
                state['content'][manifest_path] = json.dumps(manifest).encode()
                state['content'][artifact_path] = payload
                output = run(installed, ['update'])
                self.assertIn(manifest_path, state['requests'])
                self.assertNotIn(artifact_path, state['requests'])
                self.assertIn('已是最新版', output)
                self.assertEqual(installed.read_bytes(), old)
                self.assertEqual(key_path.read_text().strip(), key)

                # A download whose bytes do not match the manifest must leave
                # the old installation and synthetic key intact.
                state['content'][artifact_path] = payload[:-1] + bytes([payload[-1] ^ 1])
                output = run(installed, ['update', '--reinstall'], expected_code=1)
                self.assertIn('SHA-256', output)
                self.assertEqual(installed.read_bytes(), old)
                self.assertEqual(key_path.read_text().strip(), key)

                # Reinstallation is required for a same-version payload change.
                # It also verifies files.path resolves below latest.json.
                state['content'][artifact_path] = payload
                output = run(installed, ['update', '--reinstall'])
                self.assertIn(artifact_path, state['requests'], output)
                self.assertEqual(installed.read_bytes(), payload, output)
                self.assertEqual(key_path.read_text().strip(), key)
                record = json.loads((installed.parent / 'install-manifest.json').read_text())
                self.assertEqual(record['version'], '0.9.2')
                self.assertEqual(record['files']['tool/myautoapi-imagines-tool'], sha256(payload))
                self.assertEqual(run(installed, ['version']).strip(), '0.9.2')
                self.assertEqual(state['unexpected'], [])
                self.assertEqual(state['requests'], [manifest_path, manifest_path, artifact_path, manifest_path, artifact_path])
            finally:
                for item in (server, proxy):
                    item.shutdown()
                    item.server_close()
                for thread in threads:
                    thread.join(timeout=5)


if __name__ == '__main__':
    unittest.main(verbosity=2)
