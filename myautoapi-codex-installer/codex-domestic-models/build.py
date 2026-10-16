#!/usr/bin/env python3
"""Build standalone installers. Python is needed by maintainers only."""
import json
from pathlib import Path

ROOT = Path(__file__).resolve().parent
# Codex version whose bundled catalog is saved in src/gpt-catalog.json; update both together.
OFFICIAL_SNAPSHOT_VERSION = '0.162.1'


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
            'support_verbosity': model['slug'] == 'deepseek-flash',
            'default_verbosity': 'low' if model['slug'] == 'deepseek-flash' else None,
            # Legacy Codex uses this flag for the entire reasoning object, including effort.
            'supports_reasoning_summaries': bool(model['supported_reasoning_levels']), 'supports_reasoning_summary_parameter': False,
            'default_reasoning_summary': 'none', 'prefer_websockets': False,
            'web_search_tool_type': 'text', 'supports_search_tool': False,
            'truncation_policy': {'mode': 'tokens', 'limit': 10000},
            'supports_image_detail_original': model.get('supports_image_detail_original', model['slug'] == 'deepseek-flash'),
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
    shell_processes = parent_shell_process_functions()
    shell = (ROOT / 'src/installer.sh.in').read_text(encoding='utf-8')
    for marker, content in {
        '@MODEL_COUNT@': str(len(data['models'])),
        '# @PARENT_PROCESS_FUNCTIONS@': shell_processes,
        '// @DOMESTIC_CATALOG@': 'var DOMESTIC_CATALOG = ' + json.dumps(data, ensure_ascii=False, separators=(',', ':')) + ';',
        '// @CORE@': (ROOT / 'src/core.js').read_text(encoding='utf-8'),
        '// @MACOS_HOST@': (ROOT / 'src/macos-host.js').read_text(encoding='utf-8'),
        '// @LINUX_HOST@': (ROOT / 'src/linux-host.js').read_text(encoding='utf-8'),
    }.items():
        assert shell.count(marker) == 1
        shell = shell.replace(marker, content)
    ps = '''#Requires -Version 5.1
param([string]$CodexHomePath, [string]$CodexPath, [switch]$Uninstall)
$ErrorActionPreference = 'Stop'
function Say([string]$Message) { Write-Host $Message }
'''
    ps += '$DomesticCatalog = ConvertFrom-Json -InputObject @\'\n' + json.dumps(data, ensure_ascii=False) + "\n'@\n"
    official = json.loads((ROOT / 'src/gpt-catalog.json').read_text(encoding='utf-8'))
    ps += "$BundledOfficialVersion = '" + OFFICIAL_SNAPSHOT_VERSION + "'\n"
    ps += '$BundledOfficialCatalog = ConvertFrom-Json -InputObject @\'\n' + json.dumps(official, ensure_ascii=False) + "\n'@\n"
    ps += parent_powershell_process_functions() + '\n' + (ROOT / 'src/core.ps1').read_text(encoding='utf-8') + '\n' + (ROOT / 'src/windows-host.ps1').read_text(encoding='utf-8')
    ps += '''
try { Start-ModelInstaller $CodexHomePath $CodexPath -Uninstall:$Uninstall }
catch {
  Write-Host (Format-ModelImportError $_ -Uninstall:$Uninstall) -ForegroundColor Red
  if ($Uninstall) { throw '国产模型卸载未完成，软件未关闭。' }
  throw '国产模型导入未完成，软件未关闭。'
}
'''
    target = ROOT / 'install'
    target.mkdir(exist_ok=True)
    (target / 'codex-models.sh').write_text(shell, encoding='utf-8', newline='\n')
    (target / 'codex-models.ps1').write_text(ps, encoding='utf-8-sig', newline='\n')
    return data


if __name__ == '__main__':
    data = build()
    print(f"已构建两个独立脚本，共 {len(data['models'])} 个国产模型。")
