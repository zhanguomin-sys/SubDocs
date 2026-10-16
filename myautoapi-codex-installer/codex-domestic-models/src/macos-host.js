/* macOS built-in JavaScript for Automation; no Node installation required. */
if (typeof ObjC !== 'undefined') { ObjC.import('Foundation'); }
function run(argv) {
    'use strict';
    var fm = $.NSFileManager.defaultManager, utf8 = $.NSUTF8StringEncoding;
    function nativeError(message, path, native) {
        var error = new Error(message); error.safeMessage = message;
        if (path) { error.importTarget = path; }
        try { if (native && native[0]) { var code = Number(native[0].code); if (isFinite(code)) { error.systemCode = code; } } } catch (ignored) {}
        throw error;
    }
    function attrs(path) { var error = Ref(); return fm.attributesOfItemAtPathError($(path), error); }
    function exists(path) { return !attrs(path).isNil(); }
    function isFile(path) { var value = attrs(path); return !value.isNil() && ObjC.unwrap(value.objectForKey($.NSFileType)) === ObjC.unwrap($.NSFileTypeRegular); }
    function read(path) {
        if (!isFile(path)) { nativeError('文件不存在或不是普通文件。', path); }
        var error = Ref(), value = $.NSString.stringWithContentsOfFileEncodingError($(path), utf8, error);
        if (value.isNil()) { nativeError('无法按 UTF-8 读取文件。', path, error); }
        return ObjC.unwrap(value);
    }
    function permissions(path) {
        var error = Ref();
        if (!fm.setAttributesOfItemAtPathError($.NSDictionary.dictionaryWithObjectForKey($(384), $.NSFilePosixPermissions), $(path), error)) { nativeError('无法设置文件权限。', path, error); }
    }
    function write(path, text) {
        var error = Ref();
        if (!$(text).writeToFileAtomicallyEncodingError($(path), true, utf8, error)) { nativeError('无法写入文件。', path, error); }
        permissions(path);
    }
    function remove(path) {
        if (exists(path) && !fm.removeItemAtPathError($(path), Ref())) { nativeError('无法清理暂存文件。'); }
    }
    function execute(executable, args, env) {
        var task = $.NSTask.alloc.init, outputPath = join(env.CODEX_HOME, '.myautoapi-export-' + ObjC.unwrap($.NSUUID.UUID.UUIDString));
        var errorPath = outputPath + '.stderr', output = null, errors = null;
        try {
            write(outputPath, ''); write(errorPath, '');
            output = $.NSFileHandle.fileHandleForWritingAtPath($(outputPath));
            errors = $.NSFileHandle.fileHandleForWritingAtPath($(errorPath));
            if (output.isNil() || errors.isNil()) { nativeError('无法打开后端校验输出文件。', outputPath); }
            task.executableURL = $.NSURL.fileURLWithPath($(executable)); task.arguments = $(args);
            task.standardOutput = output; task.standardError = errors;
            task.currentDirectoryURL = $.NSURL.fileURLWithPath($(env.CODEX_HOME));
            var environment = $.NSProcessInfo.processInfo.environment.mutableCopy;
            environment.setObjectForKey($(env.CODEX_HOME), $('CODEX_HOME'));
            var searchPath = ObjC.unwrap(environment.objectForKey($('PATH'))) || '';
            environment.setObjectForKey($(searchPath.split(':').map(function (entry) { return resolve(ObjC.unwrap(fm.currentDirectoryPath), entry || '.'); }).join(':')), $('PATH'));
            task.environment = environment;
            var launchError = Ref();
            if (!task.launchAndReturnError(launchError)) { nativeError('无法启动 Codex 后端。', executable, launchError); }
            var deadline = Date.now() + 30000;
            while (task.running && Date.now() < deadline) { $.NSThread.sleepForTimeInterval(0.05); }
            if (task.running) {
                task.terminate;
                $.NSThread.sleepForTimeInterval(0.2);
                if (task.running) {
                    var killer = $.NSTask.alloc.init;
                    killer.executableURL = $.NSURL.fileURLWithPath($('/bin/kill'));
                    killer.arguments = $(['-KILL', String(task.processIdentifier)]);
                    if (!killer.launchAndReturnError(Ref())) { nativeError('无法结束超时的校验后端。'); }
                    killer.waitUntilExit;
                }
                task.waitUntilExit;
                throw DomesticModels.backendFailure('', {code: 'ETIMEDOUT'});
            }
            task.waitUntilExit;
            if (task.terminationStatus !== 0) { throw DomesticModels.backendFailure(read(errorPath), {status: Number(task.terminationStatus)}); }
            return read(outputPath);
        } finally {
            if (output && !output.isNil()) { output.closeFile; }
            if (errors && !errors.isNil()) { errors.closeFile; }
            try { remove(outputPath); remove(errorPath); } catch (ignored) {}
        }
    }
    var homeValue = $(argv[0]).stringByExpandingTildeInPath.stringByStandardizingPath, executable = argv[1];
    if (!homeValue.isAbsolutePath) { homeValue = fm.currentDirectoryPath.stringByAppendingPathComponent(homeValue).stringByStandardizingPath; }
    var home = ObjC.unwrap(homeValue), action = argv[2] || 'import';
    function resolve(base, path) {
        var value = $(path).stringByExpandingTildeInPath;
        if (!value.isAbsolutePath) { value = $(base).stringByAppendingPathComponent(value); }
        return ObjC.unwrap(value.stringByStandardizingPath);
    }
    function join(a, b) { return ObjC.unwrap($(a).stringByAppendingPathComponent($(b))); }
    var io = {
        join: join, resolve: resolve, read: read, write: write, isFile: isFile, remove: remove,
        exists: exists,
        stamp: function () { return new Date().toISOString().replace(/[^0-9]/g, '') + '.' + ObjC.unwrap($.NSUUID.UUID.UUIDString); },
        mkdir: function (path, recursive) {
            if (!fm.createDirectoryAtPathWithIntermediateDirectoriesAttributesError($(path), !!recursive, $.NSDictionary.dictionaryWithObjectForKey($(448), $.NSFilePosixPermissions), Ref())) { nativeError('无法创建配置或暂存目录。'); }
        },
        backup: function (from, to) {
            if (!fm.copyItemAtPathToPathError($(from), $(to), Ref())) { nativeError('备份失败，停止导入。'); }
            permissions(to);
        },
        exportAt: function (backend, args, target) { return execute(backend, args, {CODEX_HOME: target}); },
        exportCatalog: function (backend, bundled) { return execute(backend, bundled ? ['debug', 'models', '--bundled'] : ['debug', 'models'], {CODEX_HOME: home}); },
        transaction: function (files, backend, catalogPath, original, readSet) {
            DomesticModels.transaction(io, files, backend, home, original, readSet);
        }
    };
    try {
        if (action === 'uninstall') { return DomesticModels.uninstallMessage(DomesticModels.uninstallModels(io, home)); }
        var count = DomesticModels.importModels(io, home, executable, DOMESTIC_CATALOG);
        return '已导入 ' + count + ' 个国产模型，原 GPT 和其他模型已保留。';
    } catch (error) {
        /* Never echo external process stderr, original TOML, or JSON (may contain credentials). */
        throw new Error(DomesticModels.formatError(error, action) + '\n软件未关闭。');
    }
}
