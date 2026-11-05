#!/usr/bin/env python3
"""Build the MyAutoAPI downloads from local source and checked release artifacts."""
from pathlib import Path
import hashlib
import shutil
import subprocess
import sys
import zipfile
import build
from packaging import mac_application

ROOT = Path(__file__).resolve().parents[2]
RELEASE = '0.8.3'
PUBLIC = ROOT / 'docs/public'
RELEASE_ROOT = PUBLIC / f'install/myautoapi-imagines-tool/releases/v{RELEASE}'

def main():
    subprocess.run([sys.executable, str(Path(__file__).with_name('build.py')), *sys.argv[1:]], check=True)
    tool = RELEASE_ROOT / f'myautoapi-imagines-tool-v{RELEASE}-darwin-arm64'
    checksums = {name.lstrip('*'): digest for digest, name in
                 (line.split() for line in (RELEASE_ROOT / 'SHA256SUMS').read_text().splitlines())}
    if hashlib.sha256(tool.read_bytes()).hexdigest() != checksums[tool.name]:
        raise SystemExit('生图工具校验失败，停止打包。')
    runner = '''#!/bin/bash
set -u
resource_dir="$(cd "$(dirname "$0")" && pwd)"
"$resource_dir/tool" install
result=$?
printf '\\n按回车结束，随后可关闭本窗口。'
IFS= read -r ignored </dev/tty || true
exit "$result"
'''
    mac_application(build.MAC / '2.MyAutoAPI生图工具.app', 'com.myautoapi.imagen', RELEASE,
                    {'tool': tool.read_bytes()}, runner=runner)
    download = PUBLIC / 'download'
    download.mkdir(exist_ok=True)
    for filename in [build.WINDOWS_DOWNLOAD, build.MAC_DOWNLOAD]:
        shutil.copyfile(build.DOWNLOAD / filename, download / filename)
    instructions = f'''MyAutoAPI 中转站快速导入工具

Windows x64：打开 Windows-x64/1.{build.ASSISTANT}.exe。
同一窗口可接入 Codex、Claude，安装生图工具，导入或移除国产模型。
到 https://myautoapi.com 的 API 密钥页复制完整密钥，粘贴到助手。
生图请选择专门画图分组密钥；内置生图工具 V{RELEASE}，默认 GPT Image 2.6 Flare。

Mac Apple 芯片：解压后双击 Mac-Apple芯片 下对应的 .app。
1. Codex 接入：终端中按提示使用剪贴板密钥或粘贴密钥。
2. 生图工具：按提示粘贴专门画图分组密钥。
3. 国产模型导入：保留原有 GPT 配置，按提示操作。
Mac 接入助手为终端版；不接收网页导入链接。

接入时会关闭相应客户端，请先保存任务。
教程：https://docs.myautoapi.com/quick-start
客服：https://docs.myautoapi.com/contact
'''
    output = download / 'myautoapi-tools.zip'
    with zipfile.ZipFile(output, 'w', compression=zipfile.ZIP_DEFLATED) as archive:
        exe = build.WINDOWS / f'1.{build.ASSISTANT}.exe'
        archive.write(exe, 'Windows-x64/' + exe.name)
        for path in sorted(build.MAC.rglob('*')):
            name = 'Mac-Apple芯片/' + path.relative_to(build.MAC).as_posix()
            info = zipfile.ZipInfo(name + ('/' if path.is_dir() else ''))
            info.create_system = 3
            if path.is_dir():
                mode = 0o40755
            else:
                relative = path.relative_to(build.MAC).as_posix()
                executable = '/Contents/MacOS/' in f'/{relative}' or '/Contents/Resources/' in f'/{relative}'
                mode = 0o100755 if executable else 0o100644
            info.external_attr = mode << 16
            info.compress_type = zipfile.ZIP_DEFLATED
            archive.writestr(info, b'' if path.is_dir() else path.read_bytes())
        archive.writestr('使用说明.txt', instructions.encode('utf-8'))
    for path in download.iterdir():
        if path.stat().st_size > 25 * 1024 * 1024:
            raise SystemExit(f'{path.name} 超过 Cloudflare Pages 25 MiB 限制。')
    print(f'已生成下载文件：{download}')

if __name__ == '__main__':
    main()
