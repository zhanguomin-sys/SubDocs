#!/usr/bin/env python3
"""Isolated functional checks; all writes stay under tests/.work."""
import copy
import importlib.util
import json
import os
from pathlib import Path
import shutil
import shlex
import subprocess
import uuid
import tomllib
import unittest

ROOT = Path(__file__).resolve().parents[1]
WORK = ROOT / 'tests/.work'
WORK.mkdir(exist_ok=True)
spec = importlib.util.spec_from_file_location('build', ROOT / 'build.py')
build = importlib.util.module_from_spec(spec)
spec.loader.exec_module(build)
DOMESTIC = build.build()
(WORK / 'domestic.json').write_text(json.dumps(DOMESTIC, ensure_ascii=False))
NODE = Path(os.environ.get('TEST_NODE', ROOT / '.dev-tools/node/bin/node'))
PWSH = Path(os.environ.get('TEST_PWSH', ROOT / '.dev-tools/pwsh/pwsh'))
CODEX = Path(os.environ.get('TEST_CODEX', ROOT / '.dev-tools/codex/codex-x86_64-unknown-linux-musl'))


def execute(command, **kwargs):
    env = dict(os.environ, CODEX_HOME=str(WORK), NO_COLOR='1', TERM='dumb')
    return subprocess.run([str(x) for x in command], cwd=ROOT, env=env, text=True, capture_output=True, timeout=60, **kwargs)


def model(slug, **updates):
    value = copy.deepcopy(DOMESTIC['models'][0])
    value.update(slug=slug, display_name=slug)
    value.update(updates)
    return value


GPT = model('gpt-original', supported_reasoning_levels=[{'effort': x, 'description': x} for x in ['low', 'medium', 'high', 'xhigh', 'ultra']])
APP = {'models': [GPT, model('gpt-new')], 'version': 12}


def case(config='model = "gpt-original"\nreview_model = "gpt-original"\n', **updates):
    value = dict(config=config, existing=None, app=APP, domestic=DOMESTIC, managed=[], path='/Users/测试/.codex/myautoapi-models.json')
    value.update(updates)
    return copy.deepcopy(value)


CASES = {}
def add(name, value, ok=True):
    CASES[name] = (value, ok)


add('first_import', case())
add('empty_config', case(''))
add('only_comment', case('# existing comment'))
add('preserve_full_objects', case(existing={'models': [model('gpt-original', custom={'nested': ['keep', {'a': True}]}), model('user-custom'), model('Custom'), model('custom')], 'unknown': {'keep': True}}))
add('update_replace_owned', case(existing={'models': [GPT, model('qwen3.8-max', cancelled_field=True), model('removed-domestic'), model('user-custom')]}, managed=['qwen3.8-max', 'removed-domestic']))
legacy_models = [m for m in reversed(DOMESTIC['models']) if not m['slug'].startswith('mimo-')]
add('upgrade_existing_catalog_order', case(existing={'models': [GPT] + legacy_models}, managed=[m['slug'] for m in legacy_models]))
add('existing_mimo_ids_not_duplicated', case(existing={'models': [GPT] + DOMESTIC['models'][-2:]}))
add('existing_desktop', case('model="gpt-original"\n[desktop]\nenabled-reasoning-efforts = ["ultra", # keep\n "high",]\ntheme="dark"\n[features]\nx=true\n'))
add('quoted_keys', case('"model_catalog_json" = \'old.json\' # old path\n"model"="gpt-original"\n["desktop"]\n"enabled-reasoning-efforts" = [\'ultra\',\'high\']\n'))
add('dotted_desktop', case('model="gpt-original"\ndesktop.theme="dark"\n[features]\nx=true\n'))
add('dotted_existing', case('model="gpt-original"\ndesktop.enabled-reasoning-efforts=["ultra"] # unchanged comment\n'))
add('nested_desktop', case('model="gpt-original"\n[desktop.extra]\nfoo=true\n'))
add('crlf', case('model="gpt-original"\r\n[desktop]\r\ntheme="dark"\r\n'))
add('no_final_newline', case('model="gpt-original"\n[desktop]'))
add('only_root_no_newline', case('model="gpt-original"'))
add('bom', case('\ufeffmodel="gpt-original"\n'))
add('bom_first_desktop_header', case('\ufeff[desktop]\ntheme="dark"\n'))
add('bom_first_other_header', case('\ufeff[features]\nx=true\n'))
add('windows_path', case(path='C:\\Users\\测试\\.codex\\myautoapi-models.json'))
add('path_quotes', case(path='/Users/a"b\\c/.codex/myautoapi-models.json'))
add('multiline_string', case('model="gpt-original"\ninstructions = """\n[desktop]\nmodel_catalog_json = "fake"\n# inside string\n"""\n[features]\nx=true\n'))
add('literal_multiline', case("model=\"gpt-original\"\ninstructions='''\n[desktop]\nmodel_catalog_json='fake'\n'''\n"))
add('multiline_array', case('model="gpt-original"\nother = [\n "#inside",\n "[desktop]",\n]\n'))
add('equals_in_key', case('"custom=key" = "a=b"\nmodel="gpt-original"\n'))
add('unicode_escape', case('model = "gpt-ori\\u0067inal"\n'))
add('escaped_windows_existing_path', case('model="gpt-original"\nmodel_catalog_json="C:\\\\example\\\\config.json"\n'))
add('escaped_unicode_looking_path', case('model="gpt-original"\nmodel_catalog_json="C:\\\\U00000041\\\\models.json"\n'))
add('unicode_large_escape', case('model="gpt-ori\\U00000067inal"\n'))
add('empty_desktop_array', case('[desktop]\nenabled-reasoning-efforts=[]\n'))
add('unlisted_default_alias', case('model="customer-gpt-alias"\n'))
add('unlisted_review_alias', case('model="gpt-original"\nreview_model="customer-review-alias"\n'))
add('no_gpt', case('model="user-custom"\n', app={'models':[model('user-custom')]}), False)
add('invalid_existing', case(existing={'models':[{'slug':'gpt-old'}]}), False)
add('duplicate_slugs', case(app={'models':[GPT,GPT]}), False)
add('managed_gpt', case(managed=['gpt-original']), False)
add('managed_invalid', case(managed=[123]), False)
add('unknown_effort', case('[desktop]\nenabled-reasoning-efforts=["turbo"]\n'), False)
add('effort_not_array', case('[desktop]\nenabled-reasoning-efforts="high"\n'), False)
add('effort_nonstring', case('[desktop]\nenabled-reasoning-efforts=[123]\n'), False)
add('duplicate_desktop', case('[desktop]\na=1\n[desktop]\nb=2\n'), False)
add('inline_desktop', case('desktop={theme="dark"}\n'), False)
add('profile_catalog_override', case('[profiles.work]\nmodel_catalog_json="elsewhere.json"\n'), False)
add('duplicate_catalog_key', case('model_catalog_json="a"\n"model_catalog_json"="b"\n'), False)
add('unclosed_string', case('a="bad\n'), False)
add('unclosed_multiline', case('a="""\n'), False)
add('unclosed_array', case('a=["bad"\n'), False)


class ModelTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        path = WORK / 'cases.json'
        path.write_text(json.dumps([value for value, _ in CASES.values()], ensure_ascii=False))
        cls.outputs = {}
        for platform, command in [('mac_core', [NODE, 'tests/driver.js', path]), ('windows_core', [PWSH, '-NoProfile', '-File', 'tests/driver.ps1', path])]:
            result = execute(command)
            if result.returncode:
                raise AssertionError(f'{platform}: {result.stderr}')
            cls.outputs[platform] = json.loads(result.stdout)

    def test_platform_parity(self):
        for index, name in enumerate(CASES):
            with self.subTest(name=name):
                left, right = (self.outputs[p][index] for p in ('mac_core', 'windows_core'))
                self.assertEqual(left['ok'], right['ok'])
                if left['ok']:
                    self.assertEqual(left['catalog'], right['catalog'])
                    self.assertEqual(left['efforts'], right['efforts'])
                    self.assertEqual(tomllib.loads(left['config'].lstrip('\ufeff')), tomllib.loads(right['config'].lstrip('\ufeff')))

    def test_cases(self):
        for index, (name, (value, ok)) in enumerate(CASES.items()):
            for platform, outputs in self.outputs.items():
                with self.subTest(name=name, platform=platform):
                    output = outputs[index]
                    self.assertEqual(output['ok'], ok)
                    if not ok:
                        continue
                    parsed = tomllib.loads(output['config'].lstrip('\ufeff'))
                    self.assertEqual(parsed['model_catalog_json'], value['path'])
                    self.assertEqual(len(output['managed']), len(DOMESTIC['models']))
                    ids = [x['slug'] for x in output['catalog']['models']]
                    self.assertEqual(len(ids), len(set(ids)))
                    self.assertEqual([slug for slug in ids if slug in output['managed']], [m['slug'] for m in DOMESTIC['models']])
                    self.assertIn('gpt-original', ids)
                    original = tomllib.loads(value['config'].lstrip('\ufeff'))
                    self.assertEqual(original.get('model_catalog_json'), output['sourcePath'])
                    original.pop('model_catalog_json', None)
                    if 'desktop' in original:
                        original['desktop'].pop('enabled-reasoning-efforts', None)
                        if not original['desktop']:
                            original.pop('desktop')
                    changed = copy.deepcopy(parsed)
                    changed.pop('model_catalog_json')
                    changed['desktop'].pop('enabled-reasoning-efforts')
                    if not changed['desktop']:
                        changed.pop('desktop')
                    self.assertEqual(original, changed)
                    prior = {x['slug']: x for x in (value['existing'] or {'models': []})['models']}
                    result = {x['slug']: x for x in output['catalog']['models']}
                    for slug, item in prior.items():
                        if slug not in value['managed'] and slug not in output['managed']:
                            self.assertEqual(result[slug], item)
                    for item in DOMESTIC['models']:
                        self.assertEqual(result[item['slug']], item)

    def test_exact_ids(self):
        self.assertEqual([m['slug'] for m in DOMESTIC['models']], ['deepseek-flash','glm-5.3','glm-5.3-flash','qwen3.8-max','qwen3.8-flash','kimi-k3','minimax-m3','hy4-preview','mimo-v2.6-pro','mimo-v2.6-flash'])

    def test_mimo_official_codex_capabilities(self):
        mimo_ids = {'mimo-v2.6-pro', 'mimo-v2.6-flash'}
        self.assertEqual({m['slug'] for m in DOMESTIC['models'] if m['use_responses_lite']}, mimo_ids)
        for item in DOMESTIC['models']:
            if item['slug'] not in mimo_ids:
                continue
            with self.subTest(model=item['slug']):
                self.assertEqual(item['context_window'], 1048576)
                self.assertEqual(item['input_modalities'], ['text', 'image'])
                self.assertEqual(item['default_reasoning_level'], 'low')
                self.assertEqual([level['effort'] for level in item['supported_reasoning_levels']], ['none', 'low', 'medium', 'high'])
                self.assertEqual(item['apply_patch_tool_type'], 'freeform')
                self.assertTrue(item['supports_image_detail_original'])
                self.assertFalse(item['supports_parallel_tool_calls'])

    def test_embedded_process_functions(self):
        shell, ps = build.parent_process_functions()
        self.assertIn(shell, (ROOT/'install/codex-models.sh').read_text())
        self.assertIn(ps, (ROOT/'install/codex-models.ps1').read_text(encoding='utf-8-sig'))

    def test_generated_scripts_reproducible(self):
        before = {p.name: p.read_bytes() for p in (ROOT/'install').iterdir()}
        build.build()
        self.assertEqual(before, {p.name: p.read_bytes() for p in (ROOT/'install').iterdir()})

    def test_syntax_and_platform_guard(self):
        self.assertEqual(execute(['/bin/sh','-n','install/codex-models.sh']).returncode,0)
        result = execute(['/bin/sh','-c','uname() { printf "FreeBSD\\n"; }; installer_path=$1; shift; . "$installer_path"','guard',str(ROOT/'install/codex-models.sh')])
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('macOS', result.stderr)
        result = execute([PWSH,'-NoProfile','-File','install/codex-models.ps1'])
        self.assertNotEqual(result.returncode, 0)
        self.assertIn('Windows', result.stdout)

    def linux_fixture(self, name):
        target=WORK/(name+' 用户 '+uuid.uuid4().hex)
        target.mkdir()
        bundled=json.loads(execute([CODEX,'debug','models','--bundled'],check=True).stdout)
        selected=next(m['slug'] for m in bundled['models'] if m['slug'].startswith('gpt-'))
        original=f'''# preserve existing settings
model = {json.dumps(selected)}
review_model = {json.dumps(selected)}
model_provider = "relay"
model_reasoning_effort = "high"
model_instructions_file = "instructions.md"
[model_providers.relay]
name = "relay"
base_url = "https://example.invalid/v1"
wire_api = "responses"
requires_openai_auth = false
[desktop]
theme = "dark"
'''
        (target/'config.toml').write_text(original)
        (target/'instructions.md').write_text('Test repository instructions.\n')
        (target/'auth.json').write_text('sentinel-do-not-read')
        tools=target/'bin'
        tools.mkdir()
        for name, executable in [('node',NODE),('codex',CODEX)]+[(name,Path(shutil.which(name))) for name in ('uname','awk','id','tr')]:
            (tools/name).symlink_to(executable)
        return target,tools

    def run_linux_script(self, target, tools, backend=None, piped=False, running_backend=None, shell=('/bin/sh',), env_overrides=None, platform=None, extra_args=()):
        log=(target if target.exists() else tools)/'mock-kills.log'
        log.write_text('')
        env=dict(os.environ,CODEX_HOME=str(target),PATH=str(tools),LINUX_TEST_KILL_LOG=str(log),NO_COLOR='1')
        env.update(env_overrides or {})
        wrapper='ps() { :; }; kill() { printf "%s\\n" "$*" >> "$LINUX_TEST_KILL_LOG"; }; '
        if platform:
            env['LINUX_TEST_PLATFORM']=platform
            wrapper+='uname() { printf "%s\\n" "$LINUX_TEST_PLATFORM"; }; '
        if running_backend is not None:
            env['LINUX_TEST_APP_BACKEND']=str(running_backend)
            env['LINUX_TEST_USER_UID']=str(os.getuid())
            wrapper+='ps() { if [ "$1" = -axo ]; then printf "7001 1 %s S %s\\n" "$LINUX_TEST_USER_UID" "$LINUX_TEST_APP_BACKEND"; fi; }; readlink() { [ "$1" = /proc/7001/exe ] && printf "%s\\n" "$LINUX_TEST_APP_BACKEND"; }; '
        arguments=['--codex-home',str(target)]
        if backend is not None:arguments+=['--codex-path',str(backend)]
        arguments+=list(extra_args)
        if piped:
            command=[*shell,'-c',wrapper+'. /dev/stdin','linux-test',*arguments]
            content=(ROOT/'install/codex-models.sh').read_text()
        else:
            command=[*shell,'-c',wrapper+'installer_path=$1; shift; . "$installer_path"','linux-test',str(ROOT/'install/codex-models.sh'),*arguments]
            content=None
        return subprocess.run(command,cwd=ROOT,env=env,input=content,text=True,capture_output=True,timeout=60)

    def test_linux_module_default_cannot_skip_import(self):
        target,tools=self.linux_fixture('linux-module-default')
        result=self.run_linux_script(target,tools,CODEX,piped=True,
            env_overrides={'NODE_OPTIONS':'--experimental-default-type=module'})
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        self.assertIn(f"已导入 {len(DOMESTIC['models'])} 个国产模型",result.stdout)
        for name in ('config.toml','myautoapi-models.json','myautoapi-models.managed.json'):
            self.assertTrue((target/name).is_file())
        self.assertEqual(len(json.loads((target/'myautoapi-models.managed.json').read_text())),len(DOMESTIC['models']))

    def test_linux_zero_exit_without_import_never_closes_clients(self):
        target,tools=self.linux_fixture('linux-no-receipt')
        preload=tools/'exit-before-import.js'
        preload.write_text('process.exit(0);\n')
        original=(target/'config.toml').read_bytes()
        result=self.run_linux_script(target,tools,CODEX,piped=True,running_backend=CODEX,
            env_overrides={'NODE_OPTIONS':'--require '+json.dumps(str(preload),ensure_ascii=False)})
        self.assertNotEqual(result.returncode,0)
        self.assertIn('运行器没有返回完整导入结果',result.stderr)
        self.assertEqual((target/'config.toml').read_bytes(),original)
        self.assertFalse((target/'myautoapi-models.json').exists())
        self.assertEqual((target/'mock-kills.log').read_text(),'')

    def test_linux_tries_nodejs_after_unusable_node(self):
        target,tools=self.linux_fixture('linux-nodejs-fallback')
        (tools/'node').unlink()
        (tools/'node').write_text('#!/bin/sh\nexit 1\n')
        (tools/'node').chmod(0o755)
        (tools/'nodejs').symlink_to(NODE)
        result=self.run_linux_script(target,tools,CODEX,piped=True)
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        self.assertEqual(len(json.loads((target/'myautoapi-models.managed.json').read_text())),len(DOMESTIC['models']))

    def test_linux_relative_path_keeps_node_launchers_working(self):
        target,tools=self.linux_fixture('linux-relative-path')
        (tools/'codex').unlink()
        (tools/'codex').write_text('#!/usr/bin/env node\nconst r=require("child_process").spawnSync('+json.dumps(str(CODEX))+',process.argv.slice(2),{stdio:"inherit"});process.exit(r.status===null?1:r.status);\n')
        (tools/'codex').chmod(0o755)
        relative=os.path.relpath(tools,ROOT)
        before=subprocess.run([str(tools/'codex'),'--version'],cwd=ROOT,
            env=dict(os.environ,PATH=relative),text=True,capture_output=True,timeout=30)
        self.assertEqual(before.returncode,0,before.stderr)
        result=self.run_linux_script(target,tools,piped=True,env_overrides={'PATH':relative})
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        self.assertEqual(len(json.loads((target/'myautoapi-models.managed.json').read_text())),len(DOMESTIC['models']))

    def test_macos_cli_fallback_reaches_system_runner(self):
        target,tools=self.linux_fixture('macos-cli-discovery')
        result=self.run_linux_script(target,tools,piped=True,platform='Darwin')
        # Only exercises shell discovery; this Linux host has no macOS JXA runtime.
        self.assertNotEqual(result.returncode,0)
        self.assertNotIn('未找到 Codex 后端',result.stderr)
        self.assertIn('osascript',result.stderr)
        self.assertFalse((target/'myautoapi-models.json').exists())
        self.assertEqual((target/'mock-kills.log').read_text(),'')

    def test_linux_bom_first_header_remains_valid_for_real_codex(self):
        target,tools=self.linux_fixture('linux-bom-header')
        config=target/'config.toml'
        config.write_text('\ufeff[desktop]\ntheme="dark"\n')
        env=dict(os.environ,CODEX_HOME=str(target))
        before=subprocess.run([str(CODEX),'debug','models'],cwd=target,env=env,text=True,capture_output=True,timeout=30)
        self.assertEqual(before.returncode,0,before.stderr)
        result=self.run_linux_script(target,tools,CODEX,piped=True)
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        after=subprocess.run([str(CODEX),'debug','models'],cwd=target,env=env,text=True,capture_output=True,timeout=30)
        self.assertEqual(after.returncode,0,after.stderr)
        text=config.read_text()
        self.assertTrue(text.startswith('\ufeff'))
        self.assertEqual(text.count('\ufeff'),1)
        self.assertEqual(tomllib.loads(text.lstrip('\ufeff'))['desktop']['theme'],'dark')

    def test_linux_keeps_unlisted_default_and_review_aliases(self):
        target,tools=self.linux_fixture('linux-model-alias')
        config=target/'config.toml'
        config.write_text('model="customer-gpt-alias"\nreview_model="customer-review-alias"\n')
        result=self.run_linux_script(target,tools,CODEX,piped=True)
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        after=tomllib.loads(config.read_text())
        self.assertEqual(after['model'],'customer-gpt-alias')
        self.assertEqual(after['review_model'],'customer-review-alias')
        ids=[model['slug'] for model in json.loads((target/'myautoapi-models.json').read_text())['models']]
        self.assertNotIn('customer-gpt-alias',ids)
        self.assertNotIn('customer-review-alias',ids)

    def test_linux_known_toml_parse_error_is_not_hidden_by_discovery(self):
        target,tools=self.linux_fixture('linux-duplicate-original-key')
        config=target/'config.toml'
        config.write_text('notify=[]\nnotify=[]\n')
        original=config.read_bytes()
        result=self.run_linux_script(target,tools,CODEX,piped=True)
        self.assert_linux_diagnostic(result,target,'解析已有 TOML 配置')
        self.assertIn('第 2 行',result.stderr)
        self.assertEqual(config.read_bytes(),original)
        self.assertFalse((target/'myautoapi-models.json').exists())

    def test_linux_generated_entry_and_pipe_update(self):
        target,tools=self.linux_fixture('linux-entry')
        before=tomllib.loads((target/'config.toml').read_text())
        result=self.run_linux_script(target,tools,CODEX)
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        self.assertIn(f"已导入 {len(DOMESTIC['models'])} 个国产模型",result.stdout)
        first=json.loads((target/'myautoapi-models.json').read_text())
        gpt=next(m for m in first['models'] if m['slug'].startswith('gpt-'))
        gpt['keep_metadata']={'nested':True}
        custom=copy.deepcopy(gpt);custom['slug']='user-custom'
        removed=copy.deepcopy(gpt);removed['slug']='removed-domestic'
        first['models'] += [custom,removed]
        next(m for m in first['models'] if m['slug']=='qwen3.8-max')['obsolete']=True
        (target/'myautoapi-models.json').write_text(json.dumps(first))
        (target/'myautoapi-models.managed.json').write_text(json.dumps([m['slug'] for m in DOMESTIC['models']]+['removed-domestic']))
        result=self.run_linux_script(target,tools,piped=True)
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        actual=json.loads((target/'myautoapi-models.json').read_text())
        by_id={m['slug']:m for m in actual['models']}
        self.assertNotIn('removed-domestic',by_id)
        self.assertEqual(by_id[gpt['slug']]['keep_metadata'],{'nested':True})
        self.assertEqual(by_id['user-custom'],custom)
        for model in DOMESTIC['models']:self.assertEqual(by_id[model['slug']],model)
        after=tomllib.loads((target/'config.toml').read_text())
        self.assertEqual(Path(after.pop('model_catalog_json')),target/'myautoapi-models.json')
        after['desktop'].pop('enabled-reasoning-efforts')
        self.assertEqual(after,before)
        self.assertEqual((target/'auth.json').read_text(),'sentinel-do-not-read')
        self.assertEqual((target/'mock-kills.log').read_text(),'')
        self.assertTrue(list(target.glob('config.toml.bak.*')))
        for name in ('config.toml','myautoapi-models.json','myautoapi-models.managed.json'):
            self.assertEqual((target/name).stat().st_mode & 0o777,0o600)
        self.assertFalse(list(target.glob('.myautoapi-models-*')))
        self.assertFalse(list(target.glob('.myautoapi-write-*')))

    def test_system_shell_stdin_without_bash_in_path(self):
        shells=[('/bin/sh',)]
        if shutil.which('busybox'):shells.append((shutil.which('busybox'),'ash'))
        # Previously published commands using Bash must continue to work too.
        if Path('/bin/bash').exists():shells.append(('/bin/bash',))
        for shell in shells:
            with self.subTest(shell=shell):
                target,tools=self.linux_fixture('shell-stdin')
                (tools/'ps').write_text('#!/bin/sh\nexit 0\n')
                (tools/'ps').chmod(0o755)
                self.assertIsNone(shutil.which('bash',path=str(tools)))
                before=tomllib.loads((target/'config.toml').read_text())
                env=dict(os.environ,CODEX_HOME=str(target),PATH=str(tools),NO_COLOR='1')
                result=subprocess.run([*shell,'-s','--'],input=(ROOT/'install/codex-models.sh').read_text(),
                    cwd=ROOT,env=env,text=True,capture_output=True,timeout=60)
                self.assertEqual(result.returncode,0,result.stdout+result.stderr)
                self.assertIn(f"已导入 {len(DOMESTIC['models'])} 个国产模型",result.stdout)
                after=tomllib.loads((target/'config.toml').read_text())
                self.assertEqual(after.pop('model_catalog_json'),str(target/'myautoapi-models.json'))
                after['desktop'].pop('enabled-reasoning-efforts')
                self.assertEqual(after,before)
                self.assertEqual((target/'auth.json').read_text(),'sentinel-do-not-read')

    def test_linux_import_accepts_official_and_arbitrary_providers(self):
        configs = {
            'official': '# official account; no relay configured\n[tui]\nnotifications=false\n',
            'other-provider': '''model_provider="my-company"
[model_providers.my-company]
name="my-company"
base_url="https://company.example.invalid/custom/v1"
wire_api="responses"
requires_openai_auth=false
experimental_bearer_token="private-credential-do-not-print"
''',
            'uppercase-provider': '''model_provider="OpenAI"
[model_providers.OpenAI]
name="customer-service"
base_url="https://customer.example.invalid/another/v1"
wire_api="responses"
requires_openai_auth=false
''',
            # Reproduces the user's exact backend error on the original config.
            'missing-provider': 'model_provider="OpenAI"\n',
            'missing-instructions': 'model_instructions_file="missing.md"\n',
        }
        for name,original in configs.items():
            with self.subTest(name=name):
                target,tools=self.linux_fixture('linux-provider-'+name)
                config=target/'config.toml'
                config.write_text(original)
                before=tomllib.loads(original)
                if name=='missing-provider':
                    failed=subprocess.run([str(CODEX),'debug','models'],cwd=target,
                        env=dict(os.environ,CODEX_HOME=str(target)),text=True,capture_output=True,timeout=30)
                    self.assertNotEqual(failed.returncode,0)
                    self.assertIn('Model provider `OpenAI` not found',failed.stderr)
                result=self.run_linux_script(target,tools,CODEX,piped=True)
                self.assertEqual(result.returncode,0,result.stdout+result.stderr)
                self.assertNotIn('private-credential-do-not-print',result.stdout+result.stderr)
                after=tomllib.loads(config.read_text())
                self.assertEqual(after.pop('model_catalog_json'),str(target/'myautoapi-models.json'))
                after['desktop'].pop('enabled-reasoning-efforts')
                if not after['desktop']:after.pop('desktop')
                self.assertEqual(after,before)
                self.assertEqual((target/'auth.json').read_text(),'sentinel-do-not-read')
                self.assertEqual((target/'mock-kills.log').read_text(),'')
                ids=[m['slug'] for m in json.loads((target/'myautoapi-models.json').read_text())['models']]
                self.assertTrue(any(slug.startswith('gpt-') for slug in ids))
                self.assertTrue(all(m['slug'] in ids for m in DOMESTIC['models']))

    def test_linux_import_without_existing_config_or_home(self):
        for name in ('missing-config','empty-config','missing-home'):
            with self.subTest(name=name):
                target,tools=self.linux_fixture('linux-new-'+name)
                if name=='missing-config':(target/'config.toml').unlink()
                elif name=='empty-config':(target/'config.toml').write_text('')
                else:target=target/'new'/'codex-home'
                result=self.run_linux_script(target,tools,CODEX,piped=True)
                self.assertEqual(result.returncode,0,result.stdout+result.stderr)
                config=tomllib.loads((target/'config.toml').read_text())
                self.assertEqual(set(config),{'model_catalog_json','desktop'})
                if name=='missing-home':
                    self.assertFalse((target/'auth.json').exists())
                    self.assertEqual(target.stat().st_mode & 0o777,0o700)
                else:self.assertEqual((target/'auth.json').read_text(),'sentinel-do-not-read')
                for filename in ('config.toml','myautoapi-models.json','myautoapi-models.managed.json'):
                    self.assertEqual((target/filename).stat().st_mode & 0o777,0o600)
                if name!='empty-config':self.assertFalse(list(target.glob('config.toml.bak.*')))
                ids=[m['slug'] for m in json.loads((target/'myautoapi-models.json').read_text())['models']]
                self.assertTrue(any(slug.startswith('gpt-') for slug in ids))
                self.assertTrue(all(m['slug'] in ids for m in DOMESTIC['models']))

    def test_linux_failed_validation_preserves_config_and_processes(self):
        target,tools=self.linux_fixture('linux-failure')
        config=(target/'config.toml').read_bytes()
        backend=tools/'failing-backend'
        backend.write_text('#!/bin/sh\ncase "$CODEX_HOME" in */.myautoapi-models-*) printf "private-test-value" >&2; exit 1 ;; esac\nexec '+shlex.quote(str(CODEX))+' "$@"\n')
        backend.chmod(0o755)
        result=self.run_linux_script(target,tools,backend)
        self.assertNotEqual(result.returncode,0)
        self.assertNotIn('private-test-value',result.stdout+result.stderr)
        self.assertEqual((target/'config.toml').read_bytes(),config)
        self.assertFalse((target/'myautoapi-models.json').exists())
        self.assertEqual((target/'mock-kills.log').read_text(),'')

    def test_linux_missing_runtime_does_not_write(self):
        target,tools=self.linux_fixture('linux-no-runtime')
        (tools/'node').unlink()
        config=(target/'config.toml').read_bytes()
        result=self.run_linux_script(target,tools,CODEX)
        self.assertNotEqual(result.returncode,0)
        self.assertIn('Node.js',result.stderr)
        self.assertEqual((target/'config.toml').read_bytes(),config)
        self.assertFalse((target/'myautoapi-models.json').exists())
        self.assertEqual((target/'mock-kills.log').read_text(),'')

    def test_linux_missing_codex_does_not_write(self):
        target,tools=self.linux_fixture('linux-no-codex')
        (tools/'codex').unlink()
        config=(target/'config.toml').read_bytes()
        result=self.run_linux_script(target,tools)
        self.assertNotEqual(result.returncode,0)
        self.assertIn('未找到 Codex 后端',result.stderr)
        self.assertEqual((target/'config.toml').read_bytes(),config)
        self.assertFalse((target/'myautoapi-models.json').exists())
        self.assertEqual((target/'mock-kills.log').read_text(),'')

    def test_linux_prefers_running_app_backend(self):
        target,tools=self.linux_fixture('linux-running-app')
        (tools/'codex').unlink()
        (tools/'codex').write_text('#!/bin/sh\nprintf "wrong backend" >&2\nexit 1\n')
        (tools/'codex').chmod(0o755)
        resource_dir=target/'app/resources'
        resource_dir.mkdir(parents=True)
        running_backend=resource_dir/'codex'
        running_backend.symlink_to(CODEX)
        result=self.run_linux_script(target,tools,running_backend=running_backend)
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        self.assertIn(f"已导入 {len(DOMESTIC['models'])} 个国产模型",result.stdout)
        self.assertEqual((target/'mock-kills.log').read_text().strip(),'-TERM 7001')

    def assert_linux_diagnostic(self, result, target, step, code=None):
        output=result.stdout+result.stderr
        self.assertNotEqual(result.returncode,0)
        self.assertIn('失败步骤：'+step,output)
        self.assertIn('文件或命令：',output)
        self.assertIn('原因：',output)
        self.assertIn('软件未关闭',output)
        self.assertNotIn('文件格式或系统操作失败',output)
        self.assertNotIn('private-credential-do-not-print',output)
        if code:self.assertIn(code,output)
        self.assertEqual((target/'mock-kills.log').read_text(),'')

    def test_linux_json_errors_identify_file_without_contents(self):
        for name,content,step in [
            ('myautoapi-models.managed.json','', '读取并解析已管理模型记录'),
            ('myautoapi-models.managed.json','{"key":"private-credential-do-not-print"', '读取并解析已管理模型记录'),
            ('old-models.json','private-credential-do-not-print', '读取并解析原模型目录'),
        ]:
            with self.subTest(name=name,empty=not content):
                target,tools=self.linux_fixture('linux-json-diagnostic')
                (target/name).write_text(content)
                if name=='old-models.json':
                    config=target/'config.toml'
                    config.write_text('model_catalog_json="old-models.json"\n'+config.read_text())
                before=(target/'config.toml').read_bytes()
                result=self.run_linux_script(target,tools,CODEX)
                self.assert_linux_diagnostic(result,target,step,'EJSON_PARSE')
                self.assertIn(str(target/name),result.stderr)
                self.assertEqual((target/'config.toml').read_bytes(),before)
                self.assertFalse((target/'myautoapi-models.json').exists())

    def test_linux_invalid_utf8_identifies_config(self):
        target,tools=self.linux_fixture('linux-utf8-diagnostic')
        config=target/'config.toml'
        data=config.read_bytes()+b'\xffprivate-credential-do-not-print'
        config.write_bytes(data)
        result=self.run_linux_script(target,tools,CODEX)
        self.assert_linux_diagnostic(result,target,'读取 Codex 配置','ERR_ENCODING_INVALID_ENCODED_DATA')
        self.assertIn(str(config),result.stderr)
        self.assertEqual(config.read_bytes(),data)

    def test_linux_filesystem_errors_identify_operation(self):
        target,tools=self.linux_fixture('linux-path-diagnostic')
        (target/'not-directory').write_text('private-credential-do-not-print')
        config=target/'config.toml'
        config.write_text('model_catalog_json="not-directory/models.json"\n'+config.read_text())
        before=config.read_bytes()
        result=self.run_linux_script(target,tools,CODEX)
        self.assert_linux_diagnostic(result,target,'读取并解析原模型目录','ENOTDIR')
        self.assertIn(str(target/'not-directory/models.json'),result.stderr)
        self.assertEqual(config.read_bytes(),before)

    @unittest.skipIf(os.getuid()==0,'root bypasses chmod permission restrictions')
    def test_linux_permission_errors_identify_config(self):
        target,tools=self.linux_fixture('linux-permission-diagnostic')
        config=target/'config.toml'
        before=config.read_bytes()
        config.chmod(0o000)
        try:
            result=self.run_linux_script(target,tools,CODEX)
            self.assert_linux_diagnostic(result,target,'读取 Codex 配置','EACCES')
            self.assertIn(str(config),result.stderr)
        finally:config.chmod(0o600)
        self.assertEqual(config.read_bytes(),before)

    def test_linux_backend_json_errors_identify_command(self):
        target,tools=self.linux_fixture('linux-backend-json-diagnostic')
        backend=tools/'bad-json-backend'
        backend.write_text('#!/bin/sh\nprintf "private-credential-do-not-print"\n')
        backend.chmod(0o755)
        before=(target/'config.toml').read_bytes()
        result=self.run_linux_script(target,tools,backend)
        self.assert_linux_diagnostic(result,target,'导出并解析 Codex 内置模型目录','EJSON_PARSE')
        self.assertIn(str(backend)+' debug models --bundled',result.stderr)
        self.assertEqual((target/'config.toml').read_bytes(),before)

    def test_linux_backend_errors_expose_safe_reason_and_exit_code(self):
        target,tools=self.linux_fixture('linux-backend-stderr-diagnostic')
        backend=tools/'failing-toml-backend'
        backend.write_text('#!/bin/sh\nprintf "TOML parse error at line 7, column 9\\napi_key=private-credential-do-not-print\\n" >&2\nexit 7\n')
        backend.chmod(0o755)
        before=(target/'config.toml').read_bytes()
        result=self.run_linux_script(target,tools,backend)
        self.assert_linux_diagnostic(result,target,'导出并解析 Codex 内置模型目录')
        self.assertIn('第 7 行，第 9 列',result.stderr)
        self.assertIn('后端退出码：7',result.stderr)
        self.assertEqual((target/'config.toml').read_bytes(),before)

    def test_linux_backend_missing_provider_identifies_field_without_value(self):
        target,tools=self.linux_fixture('linux-provider-diagnostic')
        backend=tools/'failing-provider-backend'
        backend.write_text("#!/bin/sh\nprintf 'Error: Model provider `private-credential-do-not-print` not found\\n' >&2\nexit 1\n")
        backend.chmod(0o755)
        before=(target/'config.toml').read_bytes()
        result=self.run_linux_script(target,tools,backend)
        self.assert_linux_diagnostic(result,target,'导出并解析 Codex 内置模型目录')
        self.assertIn('model_provider',result.stderr)
        self.assertIn('model_providers',result.stderr)
        self.assertIn('后端退出码：1',result.stderr)
        self.assertEqual((target/'config.toml').read_bytes(),before)

    def test_linux_internal_error_reports_location_without_message(self):
        source='''const core=require('./src/core.js');
const error=new TypeError('private-credential-do-not-print');
error.importStep='测试内部异常'; error.importTarget='/tmp/example-models.json';
error.stack='TypeError: private-credential-do-not-print\\n    at test ([stdin]:12:34)';
console.log(core.formatError(error));'''
        result=execute([NODE,'-e',source])
        self.assertEqual(result.returncode,0,result.stderr)
        self.assertIn('失败步骤：测试内部异常',result.stdout)
        self.assertIn('TypeError',result.stdout)
        self.assertIn('[stdin]:12:34',result.stdout)
        self.assertNotIn('private-credential-do-not-print',result.stdout)

    def test_actual_codex_deserializes_metadata(self):
        target=WORK/'actual-codex'
        target.mkdir(exist_ok=True)
        original=execute([CODEX,'debug','models','--bundled'],check=True)
        data=json.loads(original.stdout)
        gpt=[m for m in data['models'] if m['slug'].startswith('gpt-')]
        self.assertTrue(gpt)
        data['models'] += DOMESTIC['models']
        catalog=target/'models.json'
        catalog.write_text(json.dumps(data,ensure_ascii=False))
        (target/'config.toml').write_text('model_catalog_json = '+json.dumps(str(catalog))+'\n')
        env=dict(os.environ,CODEX_HOME=str(target))
        result=subprocess.run([str(CODEX),'debug','models'],cwd=target,env=env,text=True,capture_output=True,timeout=30)
        self.assertEqual(result.returncode,0,result.stderr)
        loaded=json.loads(result.stdout)
        self.assertEqual([m['slug'] for m in loaded['models']], [m['slug'] for m in data['models']])
        loaded_by_id={m['slug']: m for m in loaded['models']}
        for slug in ['mimo-v2.6-pro', 'mimo-v2.6-flash']:
            for field in ['use_responses_lite', 'supports_image_detail_original']:
                self.assertTrue(loaded_by_id[slug][field], f'{slug} {field}')
            self.assertEqual(loaded_by_id[slug]['apply_patch_tool_type'], 'freeform')

    def test_javascript_import_and_rollback(self):
        result=execute([NODE,'tests/integration.js'])
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        print(result.stdout.strip())

    def test_powershell_import_and_rollback(self):
        result=execute([PWSH,'-NoProfile','-File','tests/integration.ps1','-Backend',CODEX,'-DomesticPath',WORK/'domestic.json','-WorkRoot',WORK/'ps-integration'])
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        print(result.stdout.strip())

    def test_powershell_processes(self):
        result=execute([PWSH,'-NoProfile','-File','tests/processes.ps1','-GeneratedScript',ROOT/'install/codex-models.ps1'])
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        print(result.stdout.strip())

    def test_javascript_uninstall(self):
        result=execute([NODE,'tests/uninstall.js'])
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        print(result.stdout.strip())

    def test_powershell_uninstall(self):
        result=execute([PWSH,'-NoProfile','-File','tests/uninstall.ps1','-Backend',CODEX,'-WorkRoot',WORK/'ps-uninstall'])
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        print(result.stdout.strip())

    def test_linux_uninstall_entry_without_backend(self):
        target,tools=self.linux_fixture('uninstall-entry')
        result=self.run_linux_script(target,tools,CODEX)
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        config=(target/'config.toml').read_bytes()
        before=json.loads((target/'myautoapi-models.json').read_text())
        remaining=[m for m in before['models'] if m['slug'] not in {d['slug'] for d in DOMESTIC['models']}]
        (tools/'codex').unlink()
        result=self.run_linux_script(target,tools,piped=True,running_backend='/fake/codex',extra_args=('--uninstall',))
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        self.assertIn('已移除本工具导入的国产模型',result.stdout)
        self.assertIn('TERM', (target/'mock-kills.log').read_text())
        self.assertEqual((target/'config.toml').read_bytes(),config)
        self.assertEqual(json.loads((target/'myautoapi-models.json').read_text())['models'],remaining)
        self.assertEqual(json.loads((target/'myautoapi-models.managed.json').read_text()),[])
        loaded=subprocess.run([str(CODEX),'debug','models'],cwd=target,env=dict(os.environ,CODEX_HOME=str(target)),text=True,capture_output=True,timeout=30)
        self.assertEqual(loaded.returncode,0,loaded.stderr)
        self.assertEqual([m['slug'] for m in json.loads(loaded.stdout)['models']],[m['slug'] for m in remaining])
        before={p.name:p.read_bytes() for p in target.iterdir() if p.is_file() and p.name!='mock-kills.log'}
        result=self.run_linux_script(target,tools,piped=True,running_backend='/fake/codex',extra_args=('--uninstall',))
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        self.assertIn('没有需要移除',result.stdout)
        self.assertEqual((target/'mock-kills.log').read_text(),'')
        self.assertEqual({p.name:p.read_bytes() for p in target.iterdir() if p.is_file() and p.name!='mock-kills.log'},before)

    def test_linux_uninstall_stdin_command_and_inactive_catalog(self):
        target,tools=self.linux_fixture('uninstall-stdin')
        result=self.run_linux_script(target,tools,CODEX)
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        external=target/'my-models.json'
        external.write_bytes((target/'myautoapi-models.json').read_bytes())
        config=(target/'config.toml').read_text().replace(str(target/'myautoapi-models.json'),str(external))
        (target/'config.toml').write_text(config)
        before=external.read_bytes()
        (tools/'codex').unlink()
        (tools/'ps').write_text('#!/bin/sh\nexit 0\n'); (tools/'ps').chmod(0o755)
        env=dict(os.environ,CODEX_HOME=str(target),PATH=str(tools))
        result=subprocess.run(['/bin/sh','-s','--','--uninstall'],input=(ROOT/'install/codex-models.sh').read_text(),cwd=ROOT,env=env,text=True,capture_output=True,timeout=60)
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        self.assertIn('已清理原导入目录；软件未关闭',result.stdout)
        self.assertEqual(external.read_bytes(),before)
        self.assertEqual((target/'config.toml').read_text(),config)

    def test_linux_uninstall_bad_record_never_closes_clients(self):
        target,tools=self.linux_fixture('uninstall-invalid')
        (target/'myautoapi-models.managed.json').write_text('{"key":"private-do-not-print"}')
        config=(target/'config.toml').read_bytes()
        result=self.run_linux_script(target,tools,piped=True,running_backend='/fake/codex',extra_args=('--uninstall',))
        self.assertNotEqual(result.returncode,0)
        self.assertIn('卸载失败',result.stderr)
        self.assertIn('校验已管理模型记录',result.stderr)
        self.assertNotIn('private-do-not-print',result.stdout+result.stderr)
        self.assertEqual((target/'mock-kills.log').read_text(),'')
        self.assertEqual((target/'config.toml').read_bytes(),config)

    def test_linux_uninstall_missing_home_does_not_create_it(self):
        target,tools=self.linux_fixture('uninstall-no-home')
        missing=target/'never-created'
        result=self.run_linux_script(missing,tools,piped=True,running_backend='/fake/codex',extra_args=('--uninstall',))
        self.assertEqual(result.returncode,0,result.stdout+result.stderr)
        self.assertIn('没有需要移除',result.stdout)
        self.assertFalse(missing.exists())
        self.assertEqual((tools/'mock-kills.log').read_text(),'')

    def test_shell_processes(self):
        functions=build.parent_process_functions()[0]
        fixture='''
id() { printf '1001\\n'; }
ps() {
  if [ "$1" = -axo ]; then
    cat <<EOF
$$ $PPID 1001 S /Applications/Codex.app/Contents/MacOS/Codex
$PPID 1 1001 S codex
7001 1 1001 S /Applications/ChatGPT.app/Contents/MacOS/ChatGPT
7002 1 1002 S codex
7003 1 1001 Z codex
7004 1 1001 S codex-helper
7005 1 1001 S /Applications/Codex.app/Contents/Resources/codex
1 0 0 S launchd
EOF
  else
    return 0
  fi
}
say() { printf '%s\\n' "$*"; }
sleep() { :; }
kill() { printf '%s\\n' "$*" >> "$process_log"; }
selection="$(list_running_clients)"
case "$selection" in *"$$ codex skip"*) ;; *) exit 11 ;; esac
case "$selection" in *"$PPID codex skip"*) ;; *) exit 12 ;; esac
case "$selection" in *"7001 chatgpt stop"*) ;; *) exit 13 ;; esac
case "$selection" in *"7005 codex stop"*) ;; *) exit 14 ;; esac
case "$selection" in *7002*|*7003*|*7004*) exit 15 ;; esac
stop_running_clients
[ "$codex_running" = true ] && [ "$chatgpt_running" = true ] && [ "$clients_left" = true ] || exit 16
report_running_clients
list_running_clients() { printf '7001 codex stop\\n'; }
stop_running_clients
[ "$clients_left" = false ] || exit 17
report_running_clients
list_running_clients() { :; }
stop_running_clients
[ "$codex_running" = false ] && [ "$chatgpt_running" = false ] || exit 18
report_running_clients
list_running_clients() { printf '7001 codex stop\\n'; }
client_is_alive() { return 0; }
stop_running_clients
[ "$clients_left" = true ] || exit 19
report_running_clients
'''
        script=WORK/'processes.sh'
        process_log=WORK/'mock-kills.log'
        process_log.write_text('')
        script.write_text('set -eu\nprocess_log="$1"\n'+functions+'\n'+fixture)
        shells=[('/bin/sh',)]
        if shutil.which('busybox'):shells.append((shutil.which('busybox'),'ash'))
        for shell in shells:
            with self.subTest(shell=shell):
                process_log.write_text('')
                result=execute([*shell,script,process_log])
                self.assertEqual(result.returncode,0,result.stdout+result.stderr)
                self.assertIn('已自动关闭',result.stdout)
                self.assertIn('未能自动关闭',result.stdout)
                self.assertIn('当前未检测到',result.stdout)
                calls=process_log.read_text().splitlines()
                self.assertTrue(any('-KILL' in call for call in calls))
                for call in calls:
                    self.assertTrue(set(call.split()[1:]).issubset({'7001','7005'}))


if __name__ == '__main__':
    unittest.main(verbosity=2)
