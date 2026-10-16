'use strict';
const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const assert = require('assert/strict');
const core = require('../src/core.js');
const createIO = require('../src/linux-host.js').createLinuxIO;
const work = path.join(__dirname, '.work', 'uninstall-js');
const domestic = JSON.parse(fs.readFileSync(path.join(__dirname, '.work', 'domestic.json'), 'utf8'));
fs.mkdirSync(work, {recursive: true});
let checks = 0;
function check(value) { assert.ok(value); checks++; }
function fixture(options = {}) {
    const home = path.join(work, crypto.randomUUID()); fs.mkdirSync(home);
    const catalogPath = path.join(home, 'myautoapi-models.json');
    const gpt = {...domestic.models[0], slug: 'gpt-customer', custom: {preserve: ['full', 'metadata']}};
    const user = {...gpt, slug: 'user-custom'};
    const retired = {...gpt, slug: 'retired-owned-model'};
    const catalog = {version: 'preserve', unknown: {keep: true}, models: [gpt, user, retired, ...domestic.models]};
    fs.writeFileSync(catalogPath, JSON.stringify(catalog));
    fs.writeFileSync(path.join(home, 'myautoapi-models.managed.json'), JSON.stringify(['retired-owned-model', ...domestic.models.map(m => m.slug)]));
    const config = '\uFEFF# preserve all settings\r\nmodel="customer-alias"\r\nreview_model="custom-review"\r\nmodel_provider="custom"\r\nmodel_catalog_json='+JSON.stringify(options.external || catalogPath)+'\r\n[model_providers.custom]\r\nname="OpenAI"\r\nbase_url="https://customer.example.invalid/v1"\r\nsupports_websockets=true\r\nrequires_openai_auth=true\r\n[desktop]\r\nenabled-reasoning-efforts=["ultra","max","high"]\r\nfollowUpQueueMode="steer"\r\n';
    if (!options.noConfig) { fs.writeFileSync(path.join(home, 'config.toml'), config); }
    fs.writeFileSync(path.join(home, 'auth.json'), 'private-auth-sentinel');
    const io = createIO(home, core), rawRead = io.read;
    io.read = file => { assert.notEqual(path.basename(file), 'auth.json'); checks++; return rawRead(file); };
    io.exportAt = io.exportCatalog = () => { throw Error('uninstall must not invoke Codex'); };
    return {home, catalogPath, gpt, user, catalog, config, io};
}
function snapshot(home) { return new Map(fs.readdirSync(home).map(name => [name, fs.readFileSync(path.join(home, name))])); }
function equal(actual, expected) { assert.deepEqual(actual, expected); checks++; }
{
    const f = fixture();
    const result = core.uninstallModels(f.io, f.home);
    equal(result.count, domestic.models.length + 1); check(result.active);
    const after = JSON.parse(fs.readFileSync(f.catalogPath));
    equal(after, {...f.catalog, models: [f.gpt, f.user]});
    equal(fs.readFileSync(path.join(f.home, 'config.toml'), 'utf8'), f.config);
    equal(fs.readFileSync(path.join(f.home, 'auth.json'), 'utf8'), 'private-auth-sentinel');
    equal(JSON.parse(fs.readFileSync(path.join(f.home, 'myautoapi-models.managed.json'))), []);
    check(fs.readdirSync(f.home).some(n => n.startsWith('myautoapi-models.json.bak.')));
    check(!fs.readdirSync(f.home).some(n => n.startsWith('config.toml.bak.')));
    const before = snapshot(f.home);
    check(!core.uninstallModels(f.io, f.home).active);
    equal(snapshot(f.home), before);
}
{
    const external = path.join(work, 'external-' + crypto.randomUUID() + '.json');
    fs.writeFileSync(external, 'external catalogue must remain byte-identical');
    const f = fixture({external});
    check(!core.uninstallModels(f.io, f.home).active);
    equal(fs.readFileSync(external, 'utf8'), 'external catalogue must remain byte-identical');
    equal(fs.readFileSync(path.join(f.home, 'config.toml'), 'utf8'), f.config);
}
{
    const f = fixture({noConfig: true});
    check(!core.uninstallModels(f.io, f.home).active);
    check(!fs.existsSync(path.join(f.home, 'config.toml')));
}
for (const mode of ['absent', 'empty', 'invalid', 'gpt', 'missing-catalog', 'invalid-catalog', 'symlink-catalog']) {
    const f = fixture(), record = path.join(f.home, 'myautoapi-models.managed.json');
    if (mode === 'absent') { fs.unlinkSync(record); }
    if (mode === 'empty') { fs.writeFileSync(record, '[]'); }
    if (mode === 'invalid') { fs.writeFileSync(record, '{"credential":"must not print"}'); }
    if (mode === 'gpt') { fs.writeFileSync(record, '["gpt-customer"]'); }
    if (mode === 'missing-catalog') { fs.unlinkSync(f.catalogPath); }
    if (mode === 'invalid-catalog') { fs.writeFileSync(f.catalogPath, '{'); }
    if (mode === 'symlink-catalog') { const target = f.catalogPath + '.actual'; fs.renameSync(f.catalogPath, target); fs.symlinkSync(target, f.catalogPath); }
    const before = snapshot(f.home);
    if (mode === 'absent' || mode === 'empty') { check(!core.uninstallModels(f.io, f.home).changed); }
    else { assert.throws(() => core.uninstallModels(f.io, f.home)); checks++; }
    equal(snapshot(f.home), before);
}
for (const mode of ['write', 'external-edit', 'external-before-stage', 'readback']) {
    const f = fixture(), before = snapshot(f.home);
    const rawWrite = f.io.write; let once = false;
    f.io.write = (file, text) => {
        if (!once && mode === 'external-before-stage' && path.basename(file) === 'file-0') {
            once = true; fs.appendFileSync(path.join(f.home, 'config.toml'), '# App saved\n');
        }
        if (!once && path.basename(file) === 'myautoapi-models.managed.json') {
            once = true;
            if (mode === 'external-edit') { fs.appendFileSync(path.join(f.home, 'config.toml'), '# App saved\n'); }
            if (mode === 'readback') { rawWrite(file, text); fs.appendFileSync(f.catalogPath, ' '); return; }
            throw Error('injected write failure');
        }
        rawWrite(file, text);
    };
    assert.throws(() => core.uninstallModels(f.io, f.home)); checks++;
    const savedConfig = fs.readFileSync(path.join(f.home, 'config.toml'), 'utf8');
    equal(savedConfig, mode.startsWith('external') ? f.config + '# App saved\n' : f.config);
    if (mode === 'write' || mode === 'external-before-stage') { equal(fs.readFileSync(f.catalogPath), before.get('myautoapi-models.json')); }
    else { check(JSON.parse(fs.readFileSync(f.catalogPath)).models.every(m => !domestic.models.some(d => d.slug === m.slug))); }
    check(!fs.readdirSync(f.home).some(n => n.startsWith('.myautoapi-models-')));
}
console.log(`JavaScript uninstall: ${checks} assertions passed`);
