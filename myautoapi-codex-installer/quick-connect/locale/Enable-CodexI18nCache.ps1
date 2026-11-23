# Local entry for the upstream -BootstrapMissing flow.
# Use the bundled core so clicking the button does not execute a remote download.
[CmdletBinding()]
param(
    [switch]$BootstrapMissing,
    [switch]$CheckOnly
)

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'

if (-not ($BootstrapMissing -or $CheckOnly)) { throw 'Use -BootstrapMissing or -CheckOnly.' }
try {
    $core = Join-Path $PSScriptRoot 'Enable-CodexI18nCurrentCache.ps1'
    & $core -TimeoutSeconds 120 -CheckOnly:$CheckOnly 3>$null 4>$null 5>$null 6>$null | Out-Null
    if ($CheckOnly) {
        [Console]::WriteLine('MYAUTOAPI_LOCALE_FOUND')
    } else {
        [Console]::WriteLine('MYAUTOAPI_LOCALE_READY')
    }
}
catch {
    $phase = $_.Exception.Data['MyAutoAPILocaleStatus']
    if ($phase -in @('BUSY', 'START_FAILED', 'RENDERER_FAILED', 'CACHE_FAILED', 'RESTART_FAILED')) {
        [Console]::WriteLine("MYAUTOAPI_LOCALE_$phase")
    }
    throw
}
