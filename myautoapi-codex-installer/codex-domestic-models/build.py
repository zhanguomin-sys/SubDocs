#!/usr/bin/env python3
"""Build standalone installers. Python is needed by maintainers only."""
import hashlib
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent
TOOL_VERSION = (ROOT / 'VERSION').read_text(encoding='utf-8').strip()
# Codex version whose bundled catalog is saved in src/gpt-catalog.json; update both together.
OFFICIAL_SNAPSHOT_VERSION = '0.170.0'


def catalog():
    definition = json.loads((ROOT / 'src/models.json').read_text(encoding='utf-8'))
    models = []
    descriptions = {'none': '关闭推理', 'minimal': '最少推理', 'low': '轻度推理', 'medium': '中度推理', 'high': '增强推理', 'xhigh': '深度推理', 'ultra': '超高推理', 'max': '最大推理'}
    for index, item in enumerate(definition['models']):
        model = dict(item)
        model['supported_reasoning_levels'] = [{'effort': effort, 'description': descriptions[effort]} for effort in model.pop('efforts')]
        model.update({
            'shell_type': 'shell_command', 'visibility': 'list', 'supported_in_api': True,
            'priority': 100 + index, 'availability_nux': None, 'upgrade': None,
            'support_verbosity': model.get('support_verbosity', False),
            'default_verbosity': model.get('default_verbosity'),
            # Legacy Codex uses this flag for the entire reasoning object, including effort.
            'supports_reasoning_summaries': bool(model['supported_reasoning_levels']), 'supports_reasoning_summary_parameter': False,
            'default_reasoning_summary': 'none', 'prefer_websockets': False,
            'web_search_tool_type': 'text', 'supports_search_tool': False,
            'truncation_policy': {'mode': 'tokens', 'limit': 10000},
            'supports_image_detail_original': model.get('supports_image_detail_original', False),
            'max_context_window': model['context_window'], 'auto_compact_token_limit': None,
            'effective_context_window_percent': 95, 'experimental_supported_tools': [],
            'base_instructions': definition['instructions'],
            'model_messages': {'instructions_template': definition['instructions']},
            'use_responses_lite': model.get('use_responses_lite', False),
        })
        models.append(model)
    return {'models': models}


def parent_shell_process_functions():
    # Reuse the actual existing installers, rather than maintaining a second implementation.
    shell = (ROOT.parent / 'install/install.sh').read_text(encoding='utf-8')
    return shell[shell.index('list_running_clients() {'):shell.index('backup_if_exists() {')].rstrip()


def parent_powershell_process_functions():
    powershell = (ROOT.parent / 'install/install.ps1').read_text(encoding='utf-8-sig')
    return powershell[powershell.index('function Get-RunningCodexClients {'):powershell.index('function Show-WritePlan {')].rstrip()


def build():
    data = catalog()
    official = json.loads((ROOT / 'src/gpt-catalog.json').read_text(encoding='utf-8'))
    shell_processes = parent_shell_process_functions()
    shell = (ROOT / 'src/installer.sh.in').read_text(encoding='utf-8')
    for marker, content in {
        '@TOOL_VERSION@': TOOL_VERSION,
        '@MODEL_COUNT@': str(len(data['models'])),
        '# @PARENT_PROCESS_FUNCTIONS@': shell_processes,
        '// @DOMESTIC_CATALOG@': 'var DOMESTIC_CATALOG = ' + json.dumps(data, ensure_ascii=False, separators=(',', ':')) + ';',
        '// @OFFICIAL_CATALOG@': 'var BUNDLED_OFFICIAL_CATALOG = ' + json.dumps(official, ensure_ascii=False, separators=(',', ':')) + ';',
        '// @CORE@': (ROOT / 'src/core.js').read_text(encoding='utf-8'),
        '// @MACOS_HOST@': (ROOT / 'src/macos-host.js').read_text(encoding='utf-8'),
        '// @LINUX_HOST@': (ROOT / 'src/linux-host.js').read_text(encoding='utf-8'),
    }.items():
        assert shell.count(marker) == 1
        shell = shell.replace(marker, content)
    ps = '''#Requires -Version 5.1
param([string]$CodexHomePath, [string]$CodexPath, [switch]$Uninstall, [switch]$KeepClientsOpen, [string]$CatalogFile)
$ErrorActionPreference = 'Stop'
function Say([string]$Message) { Write-Host $Message }
'''
    ps += '$DomesticCatalog = ConvertFrom-Json -InputObject @\'\n' + json.dumps(data, ensure_ascii=False) + "\n'@\n"
    ps += "$BundledOfficialVersion = '" + OFFICIAL_SNAPSHOT_VERSION + "'\n"
    ps += '$BundledOfficialCatalog = ConvertFrom-Json -InputObject @\'\n' + json.dumps(official, ensure_ascii=False) + "\n'@\n"
    ps += parent_powershell_process_functions() + '\n' + (ROOT / 'src/core.ps1').read_text(encoding='utf-8') + '\n' + (ROOT / 'src/windows-host.ps1').read_text(encoding='utf-8')
    ps += r'''
try {
  if ($CatalogFile) {
    if ($Uninstall) { throw (New-ModelImportError '卸载不接受外部模型目录。') }
    $catalogUpdate = ConvertFrom-ModelJson (Read-PrivateText $CatalogFile) '读取新版目录' $CatalogFile
    if ($catalogUpdate.schema -ne 1 -or $catalogUpdate.official_version -cnotmatch '^[0-9]{1,4}\.[0-9]{1,4}\.[0-9]{1,4}$') {
      throw (New-ModelImportError '新版模型目录格式不兼容。')
    }
    Test-ModelCatalog $catalogUpdate.domestic
    Test-ModelCatalog $catalogUpdate.official
    $DomesticCatalog = $catalogUpdate.domestic
    $BundledOfficialCatalog = $catalogUpdate.official
    $BundledOfficialVersion = $catalogUpdate.official_version
  }
  Start-ModelInstaller $CodexHomePath $CodexPath -Uninstall:$Uninstall -KeepClientsOpen:$KeepClientsOpen
}
catch {
  Write-Host (Format-ModelImportError $_ -Uninstall:$Uninstall) -ForegroundColor Red
  if ($Uninstall) { throw '国产模型卸载未完成，软件未关闭。' }
  throw '国产模型导入未完成，软件未关闭。'
}
'''
    public = ROOT.parents[1] / 'docs' / 'public' / 'install' / 'codex-models'
    relative = f'releases/v{TOOL_VERSION}/catalog.json'
    release_file = public / relative
    release_file.parent.mkdir(parents=True, exist_ok=True)
    payload = json.dumps({
        'schema': 1, 'version': TOOL_VERSION, 'official_version': OFFICIAL_SNAPSHOT_VERSION,
        'domestic': data, 'official': official,
    }, ensure_ascii=False, separators=(',', ':')).encode('utf-8') + b'\n'
    if release_file.exists() and release_file.read_bytes() != payload:
        raise SystemExit('Model catalog changed: bump VERSION before publishing')
    # PowerShell 5.1 needs a BOM for local files. The web entry serves UTF-8
    # text without a BOM; keep both copies generated from the same source.
    for target, windows_encoding in (
        (ROOT / 'install', 'utf-8-sig'),
        (ROOT.parent / 'install', 'utf-8-sig'),
        (public.parent, 'utf-8'),
    ):
        target.mkdir(exist_ok=True)
        (target / 'codex-models.sh').write_text(shell, encoding='utf-8', newline='\n')
        (target / 'codex-models.ps1').write_text(ps, encoding=windows_encoding, newline='\n')
    release_file.write_bytes(payload)
    (public / 'latest.json').write_text(json.dumps({
        'version': TOOL_VERSION, 'path': relative, 'size': len(payload),
        'sha256': hashlib.sha256(payload).hexdigest(),
    }, indent=2) + '\n', encoding='utf-8')
    return data


if __name__ == '__main__':
    data = build()
    print(f"已构建两个独立脚本，共 {len(data['models'])} 个国产模型。")
