# Adapted from zhenglei410/codex-i18n-cache-seeder (MIT, see LICENSE).
# Upstream current-key/Enable-CodexI18nCurrentCache.ps1
# Gitee blob 00f607fdfa6116bc45500e7acbcca58f23e5c3d7
# SHA-256 335c1d49b9fa8fd35d5276bf8fae93a6bc5e8bae879d156fb52c9cd7a2b3202b
# The page expression, reload verification, and delayed reads come from
# upstream. This entry uses the assistant's scoped Windows process handling.

[CmdletBinding()]
param(
    [ValidateRange(5, 300)]
    [int]$TimeoutSeconds = 120,
    [switch]$CheckOnly
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

function Get-CodexDesktopProcesses {
    Get-CimInstance Win32_Process -Filter "SessionId = $sessionId AND Name = '$($installation.ProcessName)'" |
        Where-Object { $_.ExecutablePath -eq $installation.ExecutablePath }
}

function Get-CodexMainProcesses {
    Get-CodexDesktopProcesses | Where-Object {
        $_.CommandLine -and
        $_.CommandLine -notmatch '--type=' -and
        $_.CommandLine -notmatch 'crashpad-handler'
    }
}

function Find-FreePort {
    # Keep the working command's default port when it is available.
    foreach ($candidate in @(9240, 0)) {
        $listener = [Net.Sockets.TcpListener]::new([Net.IPAddress]::Loopback, $candidate)
        $listener.ExclusiveAddressUse = $true
        try {
            $listener.Start()
            return $listener.LocalEndpoint.Port
        }
        catch [Net.Sockets.SocketException] {
            if ($candidate -eq 0) { throw }
        }
        finally {
            $listener.Stop()
        }
    }
}

function Add-AppActivationInterop {
    if ('CodexCurrentCache.NativeActivation' -as [type]) {
        return
    }

    Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;

namespace CodexCurrentCache
{
    public static class NativeActivation
    {
        private const uint CLSCTX_LOCAL_SERVER = 0x4;

        [DllImport("ole32.dll")]
        private static extern int CoCreateInstance(
            ref Guid rclsid,
            IntPtr pUnkOuter,
            uint dwClsContext,
            ref Guid riid,
            out IntPtr ppv);

        [UnmanagedFunctionPointer(CallingConvention.Winapi)]
        private delegate int ActivateApplicationDelegate(
            IntPtr self,
            [MarshalAs(UnmanagedType.LPWStr)] string appUserModelId,
            [MarshalAs(UnmanagedType.LPWStr)] string arguments,
            int options,
            out uint processId);

        public static int Activate(
            string appUserModelId,
            string arguments,
            out uint processId)
        {
            Guid clsid = new Guid("45BA127D-10A8-46EA-8AB7-56EA9078943C");
            Guid iid = new Guid("2e941141-7f97-4756-ba1d-9decde894a3d");
            IntPtr manager;
            int hr = CoCreateInstance(
                ref clsid,
                IntPtr.Zero,
                CLSCTX_LOCAL_SERVER,
                ref iid,
                out manager);
            if (hr < 0)
            {
                processId = 0;
                return hr;
            }

            try
            {
                IntPtr vtable = Marshal.ReadIntPtr(manager);
                IntPtr method = Marshal.ReadIntPtr(vtable, 3 * IntPtr.Size);
                ActivateApplicationDelegate activate =
                    (ActivateApplicationDelegate)Marshal.GetDelegateForFunctionPointer(
                        method,
                        typeof(ActivateApplicationDelegate));
                return activate(manager, appUserModelId, arguments, 0, out processId);
            }
            finally
            {
                Marshal.Release(manager);
            }
        }
    }
}
'@
}

function Get-CodexInstallation {
    $package = Get-AppxPackage -Name OpenAI.Codex |
        Sort-Object Version -Descending |
        Select-Object -First 1

    if ($null -eq $package) {
        throw 'The OpenAI.Codex MSIX package is not installed for the current user.'
    }

    $manifestPath = Join-Path $package.InstallLocation 'AppxManifest.xml'
    [xml]$manifest = Get-Content -Raw -LiteralPath $manifestPath
    $applications = @($manifest.Package.Applications.Application)
    $application = $applications |
        Where-Object { $_.Id -eq 'App' } |
        Select-Object -First 1

    if ($null -eq $application -or -not $application.Executable) {
        throw 'The Codex desktop application entry could not be found.'
    }

    $executablePath = [IO.Path]::GetFullPath((Join-Path $package.InstallLocation $application.Executable))
    return [pscustomobject]@{
        Aumid = "$($package.PackageFamilyName)!$($application.Id)"
        ExecutablePath = $executablePath
        ProcessName = [IO.Path]::GetFileName($executablePath)
        PackagePrefix = $package.InstallLocation.TrimEnd([char]92) + [char]92
    }
}

function Start-CodexWithDebugPort {
    param([int]$DebugPort)

    Add-AppActivationInterop
    [uint32]$processId = 0
    # A renderer reload can see cached writes before Chromium commits them to disk.
    # Use the shorter commit delay for this temporary launch before the full restart.
    $arguments = @(
        '--enable-aggressive-domstorage-flushing'
        '--remote-debugging-address=127.0.0.1'
        "--remote-debugging-port=$DebugPort"
        "--remote-allow-origins=http://127.0.0.1:$DebugPort"
    ) -join ' '
    $hr = [CodexCurrentCache.NativeActivation]::Activate(
        $installation.Aumid,
        $arguments,
        [ref]$processId
    )

    if ($hr -lt 0) {
        throw ('ApplicationActivationManager failed with HRESULT 0x{0:X8}.' -f $hr)
    }

    $script:debugProcessId = $processId
}

function Start-CodexNormally {
    Add-AppActivationInterop
    [uint32]$processId = 0
    $hr = [CodexCurrentCache.NativeActivation]::Activate(
        $installation.Aumid,
        '',
        [ref]$processId
    )

    if ($hr -lt 0) {
        throw ('ApplicationActivationManager failed with HRESULT 0x{0:X8}.' -f $hr)
    }

    $deadline = [DateTime]::UtcNow.AddSeconds(30)
    do {
        Start-Sleep -Milliseconds 300
        $running = @(Get-CodexMainProcesses | Where-Object {
            $_.CommandLine -notmatch '--remote-debugging-port='
        })
    }
    while ($running.Count -eq 0 -and [DateTime]::UtcNow -lt $deadline)

    if ($running.Count -eq 0) {
        throw 'Codex did not start after activation.'
    }
}

function Get-CodexRuntimeProcesses {
    $pattern = '^(ChatGPT|Codex|codex-command-runner|codex-code-mode-host|codex-computer-use-swift)\.exe$'
    $localPrefix = Join-Path $env:LOCALAPPDATA 'OpenAI\Codex'
    Get-CimInstance Win32_Process -Filter "SessionId = $sessionId" |
        Where-Object {
            $image = $_.ExecutablePath
            $image -and ($_.Name -match $pattern -or $image -eq $installation.ExecutablePath) -and (
                $image.StartsWith($installation.PackagePrefix, [StringComparison]::OrdinalIgnoreCase) -or
                $image.StartsWith("$localPrefix\bin\", [StringComparison]::OrdinalIgnoreCase) -or
                $image.StartsWith("$localPrefix\runtimes\", [StringComparison]::OrdinalIgnoreCase)
            )
        }
}

function Stop-CodexRuntimeProcesses {
    foreach ($process in @(Get-CodexRuntimeProcesses)) {
        $running = Get-Process -Id $process.ProcessId -ErrorAction SilentlyContinue
        if ($null -ne $running -and $running.SessionId -eq $sessionId -and
            $running.Path -eq $process.ExecutablePath) {
            try {
                Stop-Process -InputObject $running -Force -ErrorAction Stop
            }
            catch {
                # A parent process may already have stopped this child.
            }
        }
    }

    $deadline = [DateTime]::UtcNow.AddSeconds(10)
    do {
        Start-Sleep -Milliseconds 250
        $remaining = @(Get-CodexRuntimeProcesses)
    }
    while ($remaining.Count -gt 0 -and [DateTime]::UtcNow -lt $deadline)

    if ($remaining.Count -gt 0) {
        throw 'Codex processes did not stop before restart.'
    }
}

function Read-CdpEndpoint {
    param([int]$DebugPort)
    $request = [Net.WebRequest]::CreateHttp("http://127.0.0.1:$DebugPort/json/list")
    $request.Proxy = $null
    $request.AllowAutoRedirect = $false
    $request.Timeout = 1500
    $request.ReadWriteTimeout = 1500
    $response = $request.GetResponse()
    try {
        $reader = [IO.StreamReader]::new($response.GetResponseStream())
        try { return (ConvertFrom-Json -InputObject ($reader.ReadToEnd())) }
        finally { $reader.Dispose() }
    }
    finally { $response.Dispose() }
}

function Get-CdpWaitSeconds {
    param([int]$RequestedSeconds)
    $remaining = ($cacheDeadline - [DateTime]::UtcNow).TotalSeconds
    if ($remaining -le 0) { throw 'The Codex cache operation timed out.' }
    return [Math]::Min($remaining, $RequestedSeconds)
}

function Wait-CdpTarget {
    param(
        [int]$DebugPort,
        [int]$WaitSeconds
    )

    $deadline = [DateTime]::UtcNow.AddSeconds((Get-CdpWaitSeconds $WaitSeconds))
    while ([DateTime]::UtcNow -lt $deadline) {
        try {
            # Wait for the packaged app started here before using the endpoint.
            $debugProcess = Get-Process -Id $script:debugProcessId -ErrorAction SilentlyContinue
            if ($null -eq $debugProcess -or $debugProcess.SessionId -ne $sessionId -or
                $debugProcess.Path -ne $installation.ExecutablePath) {
                # A packaged launch may hand off to a different app process.
                $debugCandidates = @(Get-CodexMainProcesses | Where-Object {
                    $_.CommandLine -match "(?:^|\s)--remote-debugging-port=$DebugPort(?:\s|$)"
                })
                if ($debugCandidates.Count -eq 0) { throw 'Waiting for the Codex debugging process.' }
                $script:debugProcessId = $debugCandidates[0].ProcessId
            }
            $targets = Read-CdpEndpoint -DebugPort $DebugPort
            foreach ($target in $targets) {
                if ($null -eq $target) {
                    continue
                }

                $type = $target.PSObject.Properties['type']
                $url = $target.PSObject.Properties['url']
                $webSocket = $target.PSObject.Properties['webSocketDebuggerUrl']

                if (
                    $null -ne $type -and
                    $null -ne $url -and
                    $null -ne $webSocket -and
                    $type.Value -eq 'page' -and
                    $url.Value -eq 'app://-/index.html' -and
                    -not [string]::IsNullOrWhiteSpace([string]$webSocket.Value)
                ) {
                    return $target
                }
            }
        }
        catch {
            # Codex may still be starting.
        }

        Start-Sleep -Milliseconds 500
    }

    throw "Codex did not expose app://-/index.html within $WaitSeconds seconds."
}

function Get-WebSocketUrl {
    param([object]$Target, [int]$DebugPort)

    $uri = [Uri]$Target.webSocketDebuggerUrl
    if ($uri.Scheme -ne 'ws' -or $uri.Host -notin @('127.0.0.1','localhost','[::1]') -or
        $uri.Port -ne $DebugPort -or $uri.UserInfo -or
        $uri.AbsolutePath -notmatch '^/devtools/page/[A-Za-z0-9_-]+$') {
        throw 'The Codex target did not return a local page endpoint on the expected port.'
    }
    $builder = [UriBuilder]$uri
    $builder.Host = '127.0.0.1'
    return $builder.Uri.AbsoluteUri
}

function Invoke-CdpExpression {
    param(
        [string]$WebSocketUrl,
        [string]$Expression,
        [int]$WaitSeconds
    )

    $wait = Get-CdpWaitSeconds $WaitSeconds
    $socket = [System.Net.WebSockets.ClientWebSocket]::new()
    $socket.Options.Proxy = $null
    $timeout = [System.Threading.CancellationTokenSource]::new([TimeSpan]::FromSeconds($wait))

    try {
        [void]$socket.ConnectAsync([Uri]$WebSocketUrl, $timeout.Token).GetAwaiter().GetResult()

        $payload = @{
            id = 1
            method = 'Runtime.evaluate'
            params = @{
                expression = $Expression
                awaitPromise = $true
                returnByValue = $true
            }
        } | ConvertTo-Json -Depth 12 -Compress

        $bytes = [Text.Encoding]::UTF8.GetBytes($payload)
        [void]$socket.SendAsync(
            [ArraySegment[byte]]::new($bytes),
            [System.Net.WebSockets.WebSocketMessageType]::Text,
            $true,
            $timeout.Token
        ).GetAwaiter().GetResult()

        $buffer = [byte[]]::new(65536)
        $received = [Text.StringBuilder]::new()

        while ($true) {
            $result = $socket.ReceiveAsync(
                [ArraySegment[byte]]::new($buffer),
                $timeout.Token
            ).GetAwaiter().GetResult()

            if ($result.MessageType -eq [System.Net.WebSockets.WebSocketMessageType]::Close) {
                throw 'The Codex WebSocket closed before returning a result.'
            }

            [void]$received.Append([Text.Encoding]::UTF8.GetString($buffer, 0, $result.Count))
            if (-not $result.EndOfMessage) {
                continue
            }

            $message = $received.ToString() | ConvertFrom-Json
            [void]$received.Clear()

            $idProperty = $message.PSObject.Properties['id']
            if ($null -eq $idProperty -or [int]$idProperty.Value -ne 1) {
                continue
            }

            $errorProperty = $message.PSObject.Properties['error']
            if ($null -ne $errorProperty -and $null -ne $errorProperty.Value) {
                $errorTextProperty = $errorProperty.Value.PSObject.Properties['message']
                throw "CDP error: $([string]$errorTextProperty.Value)"
            }

            $resultProperty = $message.PSObject.Properties['result']
            $resultObject = $resultProperty.Value
            $exceptionProperty = $resultObject.PSObject.Properties['exceptionDetails']
            if ($null -ne $exceptionProperty -and $null -ne $exceptionProperty.Value) {
                $textProperty = $exceptionProperty.Value.PSObject.Properties['text']
                throw "Page script failed: $([string]$textProperty.Value)"
            }

            $container = $resultObject.PSObject.Properties['result'].Value
            return $container.PSObject.Properties['value'].Value
        }
    }
    finally {
        $timeout.Dispose()
        $socket.Dispose()
    }
}

function Confirm-CurrentCacheAfterReload {
    param(
        [int]$DebugPort,
        [string]$CurrentCacheKey,
        [int]$TimeoutSeconds
    )

    $target = Wait-CdpTarget -DebugPort $DebugPort -WaitSeconds $TimeoutSeconds
    $webSocketUrl = Get-WebSocketUrl -Target $target -DebugPort $DebugPort

    try {
        $null = Invoke-CdpExpression `
            -WebSocketUrl $webSocketUrl `
            -Expression 'location.reload(); "reload-requested"' `
            -WaitSeconds 5
    }
    catch {
        # Reload normally closes the old CDP connection.
    }

    Start-Sleep -Milliseconds 750
    $escapedKey = $CurrentCacheKey.Replace('\', '\\').Replace("'", "\'")
    $verifyExpression = @"
(() => {
    const key = '$escapedKey';
    const raw = localStorage.getItem(key);
    if (raw === null) {
        return JSON.stringify({ ok: false, reason: "cache-key-missing", key });
    }
    try {
        const wrapper = JSON.parse(raw);
        const payload = typeof wrapper.data === "string"
            ? JSON.parse(wrapper.data)
            : wrapper.data;
        const values = payload?.values;
        let value = null;
        for (const metadata of Object.values(payload?.layer_configs || {})) {
            if (!metadata || typeof metadata !== "object") continue;
            const candidates = [];
            if (metadata.v !== undefined && values) {
                candidates.push(values[String(metadata.v)]);
            }
            candidates.push(metadata.value, metadata);
            value = candidates.find(candidate =>
                candidate &&
                Object.prototype.hasOwnProperty.call(candidate, "enable_i18n")
            ) || null;
            if (value) break;
        }
        return JSON.stringify({
            ok: Boolean(value && value.enable_i18n === true),
            key,
            cachedEnableI18n: value?.enable_i18n ?? null,
            cachedLocaleSource: value?.locale_source ?? null
        });
    }
    catch (error) {
        return JSON.stringify({
            ok: false,
            reason: "cache-parse-failed",
            key,
            error: String(error)
        });
    }
})()
"@

    $lastVerification = $null
    for ($attempt = 1; $attempt -le 10; $attempt++) {
        try {
            $target = Wait-CdpTarget -DebugPort $DebugPort -WaitSeconds 5
            $webSocketUrl = Get-WebSocketUrl -Target $target -DebugPort $DebugPort
            $rawVerification = Invoke-CdpExpression `
                -WebSocketUrl $webSocketUrl `
                -Expression $verifyExpression `
                -WaitSeconds 10
            $verificationText = @($rawVerification) |
                Where-Object { $_ -is [string] } |
                Select-Object -Last 1

            if (-not [string]::IsNullOrWhiteSpace($verificationText)) {
                $lastVerification = $verificationText | ConvertFrom-Json
                if ($lastVerification.ok -eq $true) {
                    return $lastVerification
                }
            }
        }
        catch {
            # The app may still be recreating the renderer after reload.
        }

        Start-Sleep -Milliseconds 500
    }

    if ($null -ne $lastVerification) {
        throw (
            'Cache persistence verification failed. ' +
            "Cached enable_i18n: $($lastVerification.cachedEnableI18n)"
        )
    }

    throw 'Cache persistence verification failed after page reload.'
}

function Get-CurrentCacheSnapshot {
    param(
        [int]$DebugPort,
        [int]$TimeoutSeconds
    )

    $target = Wait-CdpTarget -DebugPort $DebugPort -WaitSeconds $TimeoutSeconds
    $webSocketUrl = Get-WebSocketUrl -Target $target -DebugPort $DebugPort
    $snapshotExpression = @'
(async () => {
    const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
    const deadline = Date.now() + __READY_TIMEOUT_MS__;
    const targetLayer = "72216192";
    const cachePrefix = "statsig.cached.evaluations.";

    const getRoot = () => window.__STATSIG__ || globalThis.__STATSIG__;
    const getClient = () => {
        const root = getRoot();
        if (!root) return null;
        return root.firstInstance ||
            (typeof root.instance === "function" ? root.instance() : null);
    };

    let client = getClient();
    while (
        (!client || String(client.loadingStatus || "") !== "Ready") &&
        Date.now() < deadline
    ) {
        await sleep(250);
        client = getClient();
    }

    const htmlLang = String(document.documentElement.lang || "");
    if (!client) {
        return JSON.stringify({
            ok: false,
            error: "Statsig Client did not become available.",
            currentCacheKey: null,
            currentCacheKeyExists: false,
            liveEnableI18n: null,
            allCacheKeys: [],
            htmlLang,
            statsigStatus: null
        });
    }

    const statsigStatus = String(client.loadingStatus || "");
    if (statsigStatus !== "Ready") {
        return JSON.stringify({
            ok: false,
            error: "Statsig Client did not reach Ready before the timeout.",
            currentCacheKey: null,
            currentCacheKeyExists: false,
            liveEnableI18n: null,
            allCacheKeys: [],
            htmlLang,
            statsigStatus
        });
    }

    const context = client.getContext();
    const currentCacheKey = String(
        client.dataAdapter._getCacheKey(context.user)
    );
    const layer = client.getLayer(targetLayer, {
        disableExposureLog: true
    });
    const allCacheKeys = Object.keys(localStorage)
        .filter(key => key.startsWith(cachePrefix))
        .sort();

    return JSON.stringify({
        ok: true,
        currentCacheKey,
        currentCacheKeyExists: localStorage.getItem(currentCacheKey) !== null,
        liveEnableI18n: layer ? layer.get("enable_i18n", null) : null,
        allCacheKeys,
        htmlLang,
        statsigStatus
    });
})()
'@
    $snapshotExpression = $snapshotExpression.Replace(
        '__READY_TIMEOUT_MS__',
        [string]([Math]::Min(60000, $TimeoutSeconds * 1000))
    )

    $rawSnapshot = Invoke-CdpExpression `
        -WebSocketUrl $webSocketUrl `
        -Expression $snapshotExpression `
        -WaitSeconds $TimeoutSeconds
    $cleanSnapshot = @($rawSnapshot) |
        Where-Object { $_ -is [string] } |
        Select-Object -Last 1

    if ([string]::IsNullOrWhiteSpace($cleanSnapshot)) {
        throw 'The restarted Codex page did not return a diagnostic snapshot.'
    }

    return $cleanSnapshot | ConvertFrom-Json
}

$pageExpression = @'
(async () => {
    const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
    const inspectOnly = __INSPECT_ONLY__;
    const noBootstrap = __NO_BOOTSTRAP__;
    const patchAllExisting = __PATCH_ALL_EXISTING__;
    const targetLayer = "72216192";
    const targetLayerHash = "4269166702";
    const cachePrefix = "statsig.cached.evaluations.";
    const deadline = Date.now() + __READY_TIMEOUT_MS__;

    const getRoot = () => window.__STATSIG__ || globalThis.__STATSIG__;
    const getClient = () => {
        const root = getRoot();
        if (!root) return null;
        return root.firstInstance ||
            (typeof root.instance === "function" ? root.instance() : null);
    };

    let client = getClient();
    while (
        (!client || String(client.loadingStatus || "") !== "Ready") &&
        Date.now() < deadline
    ) {
        await sleep(250);
        client = getClient();
    }

    if (!client) {
        return JSON.stringify({
            ok: false,
            error: "Statsig Client did not become available.",
            origin: location.origin,
            href: location.href,
            hasStatsigRoot: Boolean(getRoot())
        });
    }

    if (String(client.loadingStatus || "") !== "Ready") {
        return JSON.stringify({
            ok: false,
            error: "Statsig Client did not reach Ready before the timeout.",
            origin: location.origin,
            href: location.href,
            htmlLang: document.documentElement.lang,
            statsigStatus: client.loadingStatus
        });
    }

    const context = client.getContext();
    const currentCacheKey = String(client.dataAdapter._getCacheKey(context.user));

    const allCacheKeys = Object.keys(localStorage)
        .filter(key => key.startsWith(cachePrefix));
    const staleCacheKeys = allCacheKeys
        .filter(key => key !== currentCacheKey);

    const layer = client.getLayer(targetLayer, { disableExposureLog: true });
    const liveEnableI18n = layer ? layer.get("enable_i18n", null) : null;
    const currentCacheValue = localStorage.getItem(currentCacheKey);

    if (inspectOnly) {
        return JSON.stringify({
            ok: true,
            mode: "inspect",
            origin: location.origin,
            href: location.href,
            htmlLang: document.documentElement.lang,
            statsigStatus: client.loadingStatus,
            currentCacheKey,
            currentCacheKeyExists: currentCacheValue !== null,
            allCacheKeys,
            staleCacheKeys,
            patchAllExisting,
            liveEnableI18n
        });
    }

    const parseCurrentPayload = () => {
        if (currentCacheValue === null) return null;
        const wrapper = JSON.parse(currentCacheValue);
        const dataWasString = typeof wrapper.data === "string";
        const payload = dataWasString ? JSON.parse(wrapper.data) : wrapper.data;
        return { wrapper, payload, dataWasString };
    };

    let parsed = null;
    try {
        parsed = parseCurrentPayload();
    }
    catch {
        parsed = null;
    }

    let wrapper = parsed?.wrapper ?? null;
    let dataWasString = parsed?.dataWasString ?? true;
    let payload = parsed?.payload ?? null;
    let seededFrom = parsed ? "current-cache" : null;

    if (!payload && context.values && context.values.has_updates === true) {
        payload = JSON.parse(JSON.stringify(context.values));
        seededFrom = "context-values";
    }

    if (!payload && !noBootstrap) {
        payload = {
            has_updates: true,
            response_format: "init-v2",
            time: Date.now(),
            feature_gates: {},
            dynamic_configs: {},
            layer_configs: {},
            values: {},
            param_stores: {},
            sdk_flags: {},
            exposures: {}
        };
        seededFrom = "minimal-bootstrap";
    }

    if (!payload) {
        return JSON.stringify({
            ok: false,
            error: "Current cache key is missing and bootstrap is disabled.",
            currentCacheKey,
            allCacheKeys,
            staleCacheKeys,
            liveEnableI18n
        });
    }

    const hasOwn = (value, key) =>
        value !== null &&
        typeof value === "object" &&
        Object.prototype.hasOwnProperty.call(value, key);

    const patchExistingLayer = targetPayload => {
        const layerMap = targetPayload?.layer_configs;
        const values = targetPayload?.values;
        if (!layerMap || typeof layerMap !== "object") return false;

        for (const [name, metadata] of Object.entries(layerMap)) {
            if (!metadata || typeof metadata !== "object") continue;

            const isTarget = name === targetLayer || name === targetLayerHash;
            const candidates = [];
            if (metadata.v !== undefined && values && typeof values === "object") {
                candidates.push(values[String(metadata.v)]);
            }
            if (metadata.value && typeof metadata.value === "object") {
                candidates.push(metadata.value);
            }
            if (isTarget) candidates.push(metadata);

            for (const candidate of candidates) {
                if (candidate && hasOwn(candidate, "enable_i18n")) {
                    candidate.enable_i18n = true;
                    if (!hasOwn(candidate, "locale_source")) {
                        candidate.locale_source = "SYSTEM";
                    }
                    return true;
                }
            }
        }

        return false;
    };

    const insertI18nLayer = targetPayload => {
        targetPayload.has_updates = true;
        targetPayload.layer_configs = targetPayload.layer_configs || {};

        if (targetPayload.response_format === "init-v2" || targetPayload.values) {
            targetPayload.response_format = "init-v2";
            targetPayload.values = targetPayload.values || {};
            let index = 0;
            while (hasOwn(targetPayload.values, String(index))) index += 1;
            targetPayload.values[String(index)] = {
                enable_i18n: true,
                locale_source: "SYSTEM"
            };
            targetPayload.layer_configs[targetLayer] = {
                v: String(index),
                r: "codex-current-key",
                s: [],
                is_device_based: true
            };
        }
        else {
            targetPayload.layer_configs[targetLayer] = {
                enable_i18n: true,
                locale_source: "SYSTEM"
            };
        }
    };

    const findCachedI18nValue = currentPayload => {
        const layerMap = currentPayload?.layer_configs;
        const values = currentPayload?.values;
        if (!layerMap || typeof layerMap !== "object") return null;

        for (const metadata of Object.values(layerMap)) {
            if (!metadata || typeof metadata !== "object") continue;

            const candidates = [];
            if (metadata.v !== undefined && values && typeof values === "object") {
                candidates.push(values[String(metadata.v)]);
            }
            if (metadata.value && typeof metadata.value === "object") {
                candidates.push(metadata.value);
            }
            candidates.push(metadata);

            for (const candidate of candidates) {
                if (
                    candidate &&
                    hasOwn(candidate, "enable_i18n")
                ) {
                    return candidate;
                }
            }
        }

        return null;
    };

    let layerPatched = patchExistingLayer(payload);
    if (!layerPatched && !noBootstrap) {
        insertI18nLayer(payload);
        layerPatched = true;
    }

    if (!layerPatched) {
        return JSON.stringify({
            ok: false,
            error: "The current cache payload has no target Layer and bootstrap is disabled.",
            currentCacheKey,
            allCacheKeys,
            staleCacheKeys,
            liveEnableI18n
        });
    }

    wrapper = wrapper || {};
    wrapper.source = wrapper.source || "Bootstrap";
    wrapper.data = dataWasString ? JSON.stringify(payload) : payload;
    wrapper.receivedAt = Date.now();
    wrapper.stableID = context.stableID ?? wrapper.stableID ?? null;
    wrapper.fullUserHash = wrapper.fullUserHash ?? null;

    const serialized = JSON.stringify(wrapper);
    localStorage.setItem(currentCacheKey, serialized);

    const savedWrapper = JSON.parse(localStorage.getItem(currentCacheKey));
    const savedPayload = typeof savedWrapper.data === "string"
        ? JSON.parse(savedWrapper.data)
        : savedWrapper.data;
    const savedValue = findCachedI18nValue(savedPayload);

    const staleCacheResults = [];
    let staleCachesModified = 0;

    if (patchAllExisting) {
        for (const staleKey of staleCacheKeys) {
            try {
                const staleRaw = localStorage.getItem(staleKey);
                if (staleRaw === null) {
                    staleCacheResults.push({
                        key: staleKey,
                        status: "missing-during-update"
                    });
                    continue;
                }

                const staleWrapper = JSON.parse(staleRaw);
                const staleDataWasString = typeof staleWrapper.data === "string";
                const stalePayload = staleDataWasString
                    ? JSON.parse(staleWrapper.data)
                    : staleWrapper.data;
                const beforeValue = findCachedI18nValue(stalePayload);
                const wasTrue = beforeValue?.enable_i18n === true;

                let stalePatched = patchExistingLayer(stalePayload);
                let status = wasTrue ? "already-true" : "patched";

                if (!stalePatched && !noBootstrap) {
                    insertI18nLayer(stalePayload);
                    stalePatched = true;
                    status = "layer-inserted";
                }

                if (!stalePatched) {
                    staleCacheResults.push({
                        key: staleKey,
                        status: "layer-missing-bootstrap-disabled"
                    });
                    continue;
                }

                if (!wasTrue || status === "layer-inserted") {
                    staleWrapper.data = staleDataWasString
                        ? JSON.stringify(stalePayload)
                        : stalePayload;
                    staleWrapper.receivedAt = Date.now();
                    localStorage.setItem(staleKey, JSON.stringify(staleWrapper));
                    staleCachesModified += 1;
                }

                const afterValue = findCachedI18nValue(stalePayload);
                staleCacheResults.push({
                    key: staleKey,
                    status,
                    cachedEnableI18n: afterValue?.enable_i18n ?? null
                });
            }
            catch (error) {
                staleCacheResults.push({
                    key: staleKey,
                    status: "error",
                    error: String(error)
                });
            }
        }
    }

    const finalCacheKeys = Object.keys(localStorage)
        .filter(key => key.startsWith(cachePrefix))
        .sort();

    return JSON.stringify({
        ok: Boolean(savedValue && savedValue.enable_i18n === true),
        mode: "write",
        origin: location.origin,
        href: location.href,
        htmlLang: document.documentElement.lang,
        statsigStatus: client.loadingStatus,
        currentCacheKey,
        currentCacheKeyExists: currentCacheValue !== null,
        seededFrom,
        liveEnableI18n,
        cachedEnableI18n: savedValue?.enable_i18n ?? null,
        cachedLocaleSource: savedValue?.locale_source ?? null,
        allCacheKeys: finalCacheKeys,
        staleCacheKeys,
        patchAllExisting,
        staleCachesModified: staleCachesModified > 0,
        staleCacheModifiedCount: staleCachesModified,
        staleCacheResults
    });
})()
'@

$phase = 'FOUND'
$script:debugProcessId = 0
try {
    $installation = Get-CodexInstallation
    if ($CheckOnly) { return }
    $sessionId = (Get-Process -Id $PID).SessionId
    $phase = 'START_FAILED'
    if (@(Get-CodexMainProcesses).Count -gt 0) {
        $phase = 'BUSY'
        throw 'Codex is already open.'
    }

    $effectiveExpression = $pageExpression.Replace(
        '__INSPECT_ONLY__', 'false'
    ).Replace(
        '__NO_BOOTSTRAP__', 'false'
    ).Replace(
        '__PATCH_ALL_EXISTING__', 'true'
    ).Replace(
        '__READY_TIMEOUT_MS__', [string]([Math]::Min(60000, $TimeoutSeconds * 1000))
    )

    $debugPort = Find-FreePort
    $cacheDeadline = [DateTime]::UtcNow.AddSeconds(220)
    Start-CodexWithDebugPort -DebugPort $debugPort
    $phase = 'RENDERER_FAILED'
    $target = Wait-CdpTarget -DebugPort $debugPort -WaitSeconds $TimeoutSeconds
    $webSocketUrl = Get-WebSocketUrl -Target $target -DebugPort $debugPort
    $phase = 'CACHE_FAILED'
    $rawResult = Invoke-CdpExpression `
        -WebSocketUrl $webSocketUrl `
        -Expression $effectiveExpression `
        -WaitSeconds $TimeoutSeconds

    $result = $rawResult | ConvertFrom-Json
    if (-not $result.ok) {
        $errorProperty = $result.PSObject.Properties['error']
        if ($null -ne $errorProperty) { throw [string]$errorProperty.Value }
        throw 'Current-cache operation failed.'
    }

    $null = Confirm-CurrentCacheAfterReload `
        -DebugPort $debugPort `
        -CurrentCacheKey $result.currentCacheKey `
        -TimeoutSeconds $TimeoutSeconds

    # Preserve the working command's delay and sampling before stopping Codex.
    for ($check = 1; $check -le 2; $check++) {
        Start-Sleep -Seconds 3
        $null = Get-CurrentCacheSnapshot -DebugPort $debugPort -TimeoutSeconds $TimeoutSeconds
    }

    $phase = 'RESTART_FAILED'
    Stop-CodexRuntimeProcesses
    $script:debugProcessId = 0
    Start-CodexNormally
}
catch {
    $failure = $_
    if ($phase -in @('START_FAILED', 'RENDERER_FAILED', 'CACHE_FAILED', 'RESTART_FAILED')) {
        # Avoid leaving the temporary listener running or Codex closed after a failed seed.
        try {
            if ($script:debugProcessId -gt 0) {
                Stop-CodexRuntimeProcesses
            }
            if (@(Get-CodexMainProcesses).Count -eq 0) {
                Start-CodexNormally
            }
        }
        catch {
            $phase = 'RESTART_FAILED'
        }
    }
    $failure.Exception.Data['MyAutoAPILocaleStatus'] = $phase
    throw $failure
}
