'use strict';
const fs = require('fs');
const core = require('../src/core.js');
const cases = JSON.parse(fs.readFileSync(process.argv[2], 'utf8'));
const output = cases.map(test => {
    try {
        const info = core.configInfo(test.config);
        const merged = core.merge(test.existing, test.app, test.domestic, test.managed, info);
        return {ok: true, catalog: merged.catalog, managed: merged.managed, efforts: merged.efforts, sourcePath: info.values.model_catalog_json === undefined ? null : core.stringValue(info.values.model_catalog_json), config: core.patchConfig(test.config, info, test.path, merged.efforts)};
    } catch (error) { return {ok: false}; }
});
process.stdout.write(JSON.stringify(output));
