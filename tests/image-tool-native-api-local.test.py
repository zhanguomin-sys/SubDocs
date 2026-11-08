#!/usr/bin/env python3
"""Smoke-test ported Linux API routes using local HTTPS and synthetic data only."""
from __future__ import annotations

import base64
from email.parser import BytesParser
from email.policy import default
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
import json
import os
from pathlib import Path
import platform
import ssl
import struct
import subprocess
import tempfile
import threading
import unittest
import zlib

ROOT = Path(__file__).resolve().parents[1]
PROGRAM = ROOT / 'docs/public/install/myautoapi-imagines-tool/current/myautoapi-imagines-tool-linux-amd64'
API_URL = b'https://api.myautoapi.com'
GPT = 'gpt-image-2.6-flare'
BANANA = 'nano-banana-2'


def fixture_png() -> bytes:
    def chunk(kind, data):
        return struct.pack('>I', len(data)) + kind + data + struct.pack('>I', zlib.crc32(kind + data))
    return (b'\x89PNG\r\n\x1a\n' + chunk(b'IHDR', struct.pack('>IIBBBBB', 16, 16, 8, 2, 0, 0, 0))
            + chunk(b'IDAT', zlib.compress((b'\0' + b'\xff\0\0' * 16) * 16)) + chunk(b'IEND', b''))


@unittest.skipUnless(platform.system() == 'Linux' and platform.machine() in ('x86_64', 'amd64'), 'requires the Linux x64 release')
class NativeAPITest(unittest.TestCase):
    def test_models_generate_edit_and_banana_keep_their_native_routes(self):
        original = PROGRAM.read_bytes()
        png = fixture_png()
        encoded = base64.b64encode(png).decode()
        with tempfile.TemporaryDirectory(prefix='myautoapi-api-local-') as temporary:
            root = Path(temporary)
            certificate, private_key = root / 'localhost.crt', root / 'localhost.key'
            subprocess.run(['openssl', 'req', '-x509', '-newkey', 'rsa:2048', '-nodes', '-days', '1',
                '-keyout', str(private_key), '-out', str(certificate), '-subj', '/CN=127.0.0.1',
                '-addext', 'subjectAltName=IP:127.0.0.1'], check=True, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
            key = 'sk-local-synthetic-api-fixture-' + '0' * 48
            state = {'requests': [], 'unexpected': [], 'prefix': ''}

            class Handler(BaseHTTPRequestHandler):
                def do_GET(self):
                    self.respond()

                def do_POST(self):
                    self.respond()

                def respond(self):
                    body = self.rfile.read(int(self.headers.get('Content-Length', '0')))
                    path = self.path.removeprefix(state['prefix'])
                    content_type = self.headers.get('Content-Type', '')
                    request = {'method': self.command, 'path': path, 'authorized': self.headers.get('Authorization') == 'Bearer ' + key}
                    if content_type.startswith('application/json'):
                        request['json'] = json.loads(body)
                    elif content_type.startswith('multipart/form-data'):
                        message = BytesParser(policy=default).parsebytes(('Content-Type: ' + content_type + '\r\n\r\n').encode() + body)
                        request['form'] = {part.get_param('name', header='content-disposition'): part.get_payload(decode=True) for part in message.iter_parts()}
                    state['requests'].append(request)
                    if path == '/v1/models' and self.command == 'GET':
                        reply = {'object': 'list', 'data': [{'id': GPT, 'object': 'model'}, {'id': BANANA, 'object': 'model'}]}
                    elif path in ('/v1/images/generations', '/v1/images/edits') and self.command == 'POST':
                        reply = {'created': 1, 'data': [{'b64_json': encoded}]}
                    elif path == '/v1/responses' and self.command == 'POST':
                        reply = {'object': 'response', 'status': 'completed', 'output': [{'type': 'message', 'content': [
                            {'type': 'output_text', 'text': '![image](data:image/png;base64,' + encoded + ')'}]}]}
                    else:
                        state['unexpected'].append(self.command + ' ' + self.path)
                        self.send_error(404)
                        return
                    content = json.dumps(reply).encode()
                    self.send_response(200)
                    self.send_header('Content-Type', 'application/json')
                    self.send_header('Content-Length', str(len(content)))
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
                authority = f'https://127.0.0.1:{server.server_port}'
                padding = len(API_URL) - len(authority)
                self.assertGreater(padding, 0)
                state['prefix'] = '/' + 'x' * (padding - 1)
                local_url = (authority + state['prefix']).encode()
                self.assertEqual(len(API_URL), len(local_url))
                self.assertGreater(original.count(API_URL), 0)
                source = root / 'local-api-program'
                source.write_bytes(original.replace(API_URL, local_url))
                source.chmod(0o755)
                home = root / 'codex-home'
                environment = dict(os.environ, CODEX_HOME=str(home), HOME=str(root), SSL_CERT_FILE=str(certificate),
                    HTTPS_PROXY=f'http://127.0.0.1:{proxy.server_port}', HTTP_PROXY=f'http://127.0.0.1:{proxy.server_port}',
                    ALL_PROXY=f'http://127.0.0.1:{proxy.server_port}', NO_PROXY='127.0.0.1')
                for variable in ('https_proxy', 'http_proxy', 'all_proxy', 'no_proxy'):
                    environment.pop(variable, None)

                def run(program, arguments, input=''):
                    completed = subprocess.run([str(program), *arguments], input=input, text=True, capture_output=True, env=environment, timeout=30)
                    self.assertEqual(completed.returncode, 0, completed.stderr or completed.stdout)
                    return completed.stdout

                run(source, ['install', '--key-stdin'], key + '\n')
                installed = home / 'tools/myautoapi-imagines-tool/myautoapi-imagines-tool'
                models = run(installed, ['models'])
                self.assertIn(GPT, models)
                self.assertIn(BANANA, models)
                reference = root / 'reference.png'
                reference.write_bytes(png)
                commands = [
                    ['generate', '--model', GPT, '--prompt', '本机合成测试：一颗红苹果'],
                    ['edit', '--model', GPT, '--prompt', '本机合成测试：改为绿色', '--image', str(reference)],
                    ['generate', '--model', BANANA, '--prompt', '本机合成测试：一颗红苹果', '--resolution', '1k', '--aspect-ratio', '1:1'],
                ]
                for index, command in enumerate(commands):
                    output = json.loads(run(installed, [*command, '--n', '1', '--output-dir', str(root / 'images'), '--output-name', f'fixture-{index}']))
                    self.assertTrue(output['ok'], output)
                    self.assertEqual(output['output_count'], 1)
                    self.assertEqual(Path(output['output_file']).read_bytes(), png)
                requests = state['requests']
                self.assertEqual([(item['method'], item['path']) for item in requests], [
                    ('GET', '/v1/models'), ('POST', '/v1/images/generations'), ('POST', '/v1/images/edits'), ('POST', '/v1/responses')])
                self.assertTrue(all(item['authorized'] for item in requests))
                self.assertEqual(requests[1]['json']['model'], GPT)
                self.assertEqual(requests[2]['form']['model'].decode(), GPT)
                self.assertEqual(requests[3]['json']['model'], BANANA)
                self.assertEqual(state['unexpected'], [])
            finally:
                for item in (server, proxy):
                    item.shutdown()
                    item.server_close()
                for thread in threads:
                    thread.join(timeout=5)


if __name__ == '__main__':
    unittest.main(verbosity=2)
