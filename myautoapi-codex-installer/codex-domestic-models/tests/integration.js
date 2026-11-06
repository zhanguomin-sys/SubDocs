'use strict';
const fs = require('fs');
const path = require('path');
const cp = require('child_process');
const assert = require('assert/strict');
const core = require('../src/core.js');
const createLinuxIO = require('../src/linux-host.js').createLinuxIO;
const root = path.resolve(__dirname, '..');
const work = path.join(__dirname, '.work', 'js-integration');
const backend = process.env.TEST_CODEX || path.join(root, '.dev-tools/codex/codex-x86_64-unknown-linux-musl');
const domestic = JSON.parse(fs.readFileSync(path.join(__dirname, '.work/domestic.json'), 'utf8'));
fs.mkdirSync(work, {recursive: true});
let assertions = 0;
function check(value) { assert.ok(value); assertions++; }
function host(home, failure) {
    let faulted = false;
    let unreadable = false;
    const io = createLinuxIO(home, core);
    const write = io.write, read = io.read, backup = io.backup, exportAt = io.exportAt;
    function currentFiles() { return ['config.toml','myautoapi-models.json','myautoapi-models.managed.json'].map(name => fs.existsSync(path.join(home,name)) ? fs.readFileSync(path.join(home,name)) : null); }
    io.read = function(file) {
        if (unreadable && file === path.join(home, 'myautoapi-models.json')) { const error = Error('injected unreadable file'); error.code='EACCES'; throw error; }
        return read(file);
    };
    io.write = function(file, text) {
        if (failure === 'write_external' && !faulted && file === path.join(home, 'myautoapi-models.managed.json')) {
            faulted = true; fs.appendFileSync(path.join(home, 'config.toml'), '# App saved a setting\n');
            io.conflictFiles = currentFiles(); throw Error('injected write failure during external edit');
        }
        if (failure === 'write' && !faulted && file === path.join(home, 'myautoapi-models.managed.json')) { faulted = true; throw Error('injected'); }
        write(file, text);
    };
    io.backup = function(from, to) {
        if (failure === 'backup') { throw Error('injected'); }
        backup(from, to);
    };
    io.exportAt = function(exe, args, target) {
            const final = target !== home && core.stringValue(core.configInfo(fs.readFileSync(path.join(target, 'config.toml'), 'utf8')).values.model_catalog_json) === path.join(home, 'myautoapi-models.json');
            if (failure === 'validation' && target !== home) { throw Error('injected'); }
            if (failure === 'postcommit' && final) { throw Error('injected'); }
            if (failure === 'deleted_config' && final) { fs.unlinkSync(path.join(home, 'config.toml')); io.conflictFiles = currentFiles(); }
            if (failure === 'unreadable_catalog' && final) { unreadable = true; io.conflictFiles = currentFiles(); }
            if (final && ['changed_config', 'changed_managed', 'changed_catalog', 'changed_then_fail'].includes(failure)) {
                const name = {changed_config:'config.toml', changed_managed:'myautoapi-models.managed.json', changed_catalog:'myautoapi-models.json', changed_then_fail:'config.toml'}[failure];
                fs.appendFileSync(path.join(home, name), '\n');
                io.conflictFiles = currentFiles();
                if (failure === 'changed_then_fail') { throw Error('injected after external edit'); }
            }
            if (failure === 'concurrent' && target !== home) { fs.appendFileSync(path.join(home, 'config.toml'), '# edited externally\n'); }
            const text = exportAt(exe, args, target);
            if (failure === 'wrong_catalog' && target !== home) { return JSON.stringify(domestic); }
            if (failure === 'wrong_metadata' && final) { const data=JSON.parse(text); data.models.find(m=>m.slug===domestic.models[0].slug).context_window=42; return JSON.stringify(data); }
            return text;
    };
    io.exportCatalog = (exe,bundled) => io.exportAt(exe, bundled ? ['debug','models','--bundled'] : ['debug','models'], home);
    return io;
}
function setup(name) {
    const home=path.join(work,name+'-'+require('crypto').randomUUID()); fs.mkdirSync(home);
    const exported=JSON.parse(host(home).exportCatalog(backend,true));
    const selected=exported.models.find(m=>m.slug.startsWith('gpt-')).slug;
    const config=`# preserve comments\nmodel = ${JSON.stringify(selected)}\nreview_model = ${JSON.stringify(selected)}\nmodel_provider = "relay"\nmodel_reasoning_effort = "high"\n[model_providers.relay]\nname = "relay"\nbase_url = "https://example.invalid/v1"\nwire_api = "responses"\nrequires_openai_auth = false\n[desktop]\ntheme = "dark"\n`;
    fs.writeFileSync(path.join(home,'config.toml'),config);
    fs.writeFileSync(path.join(home,'auth.json'),'sentinel-do-not-read');
    return {home,config};
}
{
    const {home,config}=setup('success');
    check(core.importModels(host(home),home,backend,domestic)===domestic.models.length);
    const catalog=path.join(home,'myautoapi-models.json');
    let value=JSON.parse(fs.readFileSync(catalog,'utf8'));
    check(value.models.filter(m=>m.slug.startsWith('gpt-')).length>0);
    check(fs.readFileSync(path.join(home,'auth.json'),'utf8')==='sentinel-do-not-read');
    check(fs.readdirSync(home).some(name=>name.startsWith('config.toml.bak.')));
    const gpt=value.models.find(m=>m.slug.startsWith('gpt-')); gpt.custom_metadata={keep:true};
    value.models.find(m=>m.slug===domestic.models[0].slug).obsolete=true;
    const user={...gpt,slug:'user-custom'}; value.models.push(user,{...gpt,slug:'removed-domestic'});
    fs.writeFileSync(catalog,JSON.stringify(value));
    fs.writeFileSync(path.join(home,'myautoapi-models.managed.json'),JSON.stringify([...domestic.models.map(m=>m.slug),'removed-domestic']));
    check(core.importModels(host(home),home,backend,domestic)===domestic.models.length);
    value=JSON.parse(fs.readFileSync(catalog,'utf8'));
    check(!value.models.some(m=>m.slug==='removed-domestic'));
    check(value.models.find(m=>m.slug==='user-custom').custom_metadata.keep);
    check(value.models.find(m=>m.slug===gpt.slug).custom_metadata.keep);
    check(!('obsolete' in value.models.find(m=>m.slug===domestic.models[0].slug)));
    const after=fs.readFileSync(path.join(home,'config.toml'),'utf8');
    core.importModels(host(home),home,backend,domestic);
    check(fs.readFileSync(path.join(home,'config.toml'),'utf8')===after);
    check(config.includes('wire_api = "responses"'));
}
for (const failure of ['write','backup','validation','wrong_catalog','concurrent','postcommit','wrong_metadata','changed_config','changed_managed','changed_catalog','changed_then_fail','deleted_config','unreadable_catalog','write_external']) {
    for (const existing of [false,true]) {
        const {home,config}=setup(failure);
        if (existing) { core.importModels(host(home),home,backend,domestic); }
        const paths=['config.toml','myautoapi-models.json','myautoapi-models.managed.json'];
        const before=paths.map(name=>fs.existsSync(path.join(home,name)) ? fs.readFileSync(path.join(home,name)) : null);
        const io=host(home,failure);
        assert.throws(()=>core.importModels(io,home,backend,domestic)); assertions++;
        paths.forEach((name,index)=>{
            const filename=path.join(home,name);
            const actual=fs.existsSync(filename) ? fs.readFileSync(filename) : null;
            const expected=io.conflictFiles ? io.conflictFiles[index] : (failure==='concurrent' && index===0 ? Buffer.concat([before[index],Buffer.from('# edited externally\n')]) : before[index]);
            assert.deepEqual(actual,expected); assertions++;
        });
        check(fs.readFileSync(path.join(home,'auth.json'),'utf8')==='sentinel-do-not-read');
        check(!fs.readdirSync(home).some(name=>name.startsWith('.myautoapi-models-')));
    }
}
for (const original of [null, '']) {
    for (const failure of ['postcommit', 'changed_config']) {
        const {home} = setup('new_config_rollback');
        const filename = path.join(home, 'config.toml');
        if (original === null) { fs.unlinkSync(filename); } else { fs.writeFileSync(filename, original); }
        const io=host(home,failure);
        assert.throws(()=>core.importModels(io,home,backend,domestic)); assertions++;
        if (io.conflictFiles) {
            ['config.toml','myautoapi-models.json','myautoapi-models.managed.json'].forEach((name,index)=>{ assert.deepEqual(fs.readFileSync(path.join(home,name)),io.conflictFiles[index]); assertions++; });
        } else {
            check(fs.existsSync(filename) === (original !== null));
            if (original !== null) { check(fs.readFileSync(filename, 'utf8') === original); }
            check(!fs.existsSync(path.join(home, 'myautoapi-models.json')));
            check(!fs.existsSync(path.join(home, 'myautoapi-models.managed.json')));
        }
    }
}
{
    const {home}=setup('invalid_toml');
    fs.appendFileSync(path.join(home,'config.toml'),'this is not valid toml\n');
    assert.throws(()=>core.importModels(host(home),home,backend,domestic)); assertions++;
    check(!fs.existsSync(path.join(home,'myautoapi-models.json')));
}
for (const invalid of ['directory','symlink','dangling','managed_directory']) {
    const {home}=setup(invalid);
    const destination=path.join(home,invalid==='managed_directory' ? 'myautoapi-models.managed.json' : 'myautoapi-models.json');
    const linkTarget=path.join(home,'link-target');
    if (invalid==='directory' || invalid==='managed_directory') { fs.mkdirSync(destination); }
    else { if (invalid==='symlink') { fs.writeFileSync(linkTarget,'untouched'); } fs.symlinkSync(linkTarget,destination); }
    const before=fs.readFileSync(path.join(home,'config.toml'));
    assert.throws(()=>core.importModels(host(home),home,backend,domestic)); assertions++;
    assert.deepEqual(fs.readFileSync(path.join(home,'config.toml')),before); assertions++;
    check(fs.lstatSync(destination).isDirectory() || fs.lstatSync(destination).isSymbolicLink());
    if (invalid==='symlink') { check(fs.readFileSync(linkTarget,'utf8')==='untouched'); }
    if (invalid==='dangling') { check(!fs.existsSync(linkTarget)); }
}
{
    const {home}=setup('missing_source');
    fs.appendFileSync(path.join(home,'config.toml'),'\n');
    const filename=path.join(home,'config.toml');
    fs.writeFileSync(filename,'model_catalog_json = "missing.json"\n'+fs.readFileSync(filename,'utf8'));
    check(core.importModels(host(home),home,backend,domestic)===domestic.models.length);
}
{
    const {home}=setup('relative_instructions');
    const filename=path.join(home,'config.toml');
    fs.writeFileSync(path.join(home,'instructions.md'),'Test repository instructions.\n');
    fs.writeFileSync(filename,'model_instructions_file="instructions.md"\n'+fs.readFileSync(filename,'utf8'));
    check(core.importModels(host(home),home,backend,domestic)===domestic.models.length);
    check(fs.readFileSync(filename,'utf8').includes('model_instructions_file="instructions.md"'));
}
for (const kind of ['catalog', 'managed', 'source']) {
    for (const phase of ['bundled', 'staged']) {
        const {home} = setup('read_inputs_' + kind + '_' + phase);
        core.importModels(host(home), home, backend, domestic);
        let changedPath = path.join(home, kind === 'managed' ? 'myautoapi-models.managed.json' : 'myautoapi-models.json');
        if (kind === 'source') {
            const source = path.join(home, 'customer-models.json');
            fs.copyFileSync(changedPath, source); changedPath = source;
            const filename = path.join(home, 'config.toml');
            fs.writeFileSync(filename, fs.readFileSync(filename, 'utf8').replace(JSON.stringify(path.join(home, 'myautoapi-models.json')), JSON.stringify(source)));
        }
        const paths = ['config.toml', 'myautoapi-models.json', 'myautoapi-models.managed.json'].map(name => path.join(home, name));
        const before = paths.map(file => fs.readFileSync(file, 'utf8'));
        const io = host(home), exportAt = io.exportAt;
        let changed = null;
        io.exportAt = function(exe, args, target) {
            if (changed === null && (phase === 'bundled' ? args.includes('--bundled') : target !== home)) {
                const value = JSON.parse(fs.readFileSync(changedPath, 'utf8'));
                if (kind === 'managed') { value.push('user-managed-during-export'); }
                else { value.models.push({...value.models[0], slug:'user-added-during-export'}); }
                changed = JSON.stringify(value);
                fs.writeFileSync(changedPath, changed);
            }
            return exportAt(exe, args, target);
        };
        assert.throws(() => core.importModels(io, home, backend, domestic), /被其他程序修改/); assertions++;
        check(fs.readFileSync(changedPath, 'utf8') === changed);
        paths.forEach((file, index) => { check(fs.readFileSync(file, 'utf8') === (file === changedPath ? changed : before[index])); });
        check(!fs.readdirSync(home).some(name => name.startsWith('.myautoapi-models-')));
    }
}
console.log(`JavaScript import/rollback: ${assertions} assertions passed`);
