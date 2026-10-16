/* Linux adapter: existing Node.js only; the model/config core is shared with macOS. */
function createLinuxIO(home, core) {
    'use strict';
    var fs = require('fs'), path = require('path'), os = require('os');
    var crypto = require('crypto'), child = require('child_process');
    function failure(message) { var error = new Error(message); error.safeMessage = message; return error; }
    function exists(file) {
        try { fs.lstatSync(file); return true; }
        catch (error) { if (error.code === 'ENOENT') { return false; } throw error; }
    }
    function isFile(file) { return fs.lstatSync(file).isFile(); }
    function read(file) {
        if (!isFile(file)) { throw failure('文件不存在或不是普通文件。'); }
        return new (require('util').TextDecoder)('utf-8', {fatal: true, ignoreBOM: true}).decode(fs.readFileSync(file));
    }
    function write(file, text) {
        if (exists(file) && !isFile(file)) { throw failure('目标文件不是普通文件，停止导入。'); }
        var temp = path.join(path.dirname(file), '.myautoapi-write-' + crypto.randomUUID());
        try {
            fs.writeFileSync(temp, text, {encoding: 'utf8', flag: 'wx', mode: 384});
            fs.renameSync(temp, file);
        } finally { if (exists(temp)) { fs.unlinkSync(temp); } }
    }
    function exportAt(executable, args, target) {
        try {
            return child.execFileSync(executable, args, {
                cwd: target, env: Object.assign({}, process.env, {CODEX_HOME: target, PATH: (process.env.PATH || '').split(path.delimiter).map(function (entry) { return path.resolve(entry || '.'); }).join(path.delimiter)}),
                timeout: 30000, killSignal: 'SIGKILL', maxBuffer: 64 * 1024 * 1024,
                encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe']
            });
        } catch (error) {
            throw core.backendFailure(error.stderr, error);
        }
    }
    var io = {
        join: path.join, exists: exists, isFile: isFile, read: read, write: write,
        resolve: function (base, file) {
            if (file === '~' || file.slice(0, 2) === '~/') { file = path.join(os.homedir(), file.slice(1)); }
            return path.resolve(base, file);
        },
        remove: function (file) { fs.rmSync(file, {force: true, recursive: true}); },
        mkdir: function (directory, recursive) { fs.mkdirSync(directory, {mode: 448, recursive: !!recursive}); },
        stamp: function () { return new Date().toISOString().replace(/[^0-9]/g, '') + '.' + crypto.randomUUID(); },
        backup: function (from, to) { fs.copyFileSync(from, to, fs.constants.COPYFILE_EXCL); fs.chmodSync(to, 384); },
        exportAt: exportAt,
        exportCatalog: function (backend, bundled) { return exportAt(backend, bundled ? ['debug', 'models', '--bundled'] : ['debug', 'models'], home); },
        transaction: function (files, backend, unused, original, readSet) { core.transaction(io, files, backend, home, original, readSet); }
    };
    return io;
}

function runLinuxInstaller(argv, core, domestic) {
    'use strict';
    var action = argv[2] || 'import';
    try {
        var path = require('path'), homeArg = argv[0];
        if (homeArg === '~' || homeArg.slice(0, 2) === '~/') { homeArg = path.join(require('os').homedir(), homeArg.slice(1)); }
        var home = path.resolve(homeArg), backend = argv[1] ? path.resolve(argv[1]) : '';
        if (action === 'uninstall') {
            process.stdout.write(core.uninstallMessage(core.uninstallModels(createLinuxIO(home, core), home)) + '\n');
            return;
        }
        var count = core.importModels(createLinuxIO(home, core), home, backend, domestic);
        process.stdout.write('已导入 ' + count + ' 个国产模型，原 GPT 和其他模型已保留。\n');
    } catch (error) {
        process.stderr.write(core.formatError(error, action) + '\n软件未关闭。\n');
        process.exitCode = 1;
    }
}

if (typeof module !== 'undefined' && module.exports) {
    module.exports = {createLinuxIO: createLinuxIO, runLinuxInstaller: runLinuxInstaller};
    if (typeof DOMESTIC_CATALOG !== 'undefined') { runLinuxInstaller(process.argv.slice(2), DomesticModels, DOMESTIC_CATALOG); }
}
