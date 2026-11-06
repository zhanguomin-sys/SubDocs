param([string]$SourceUrl, [string]$WorkRoot)
$ErrorActionPreference='Stop'
$target=Join-Path $WorkRoot ([Guid]::NewGuid().ToString('N'))
$null=[IO.Directory]::CreateDirectory($target)
$env:OS='Windows_NT'
$env:CODEX_HOME=$target
# The documented invocation must succeed without any CLI, executable or App path.
$env:PATH=$target
function Get-AppxPackage { throw 'Must not locate an App backend' }
# No Codex/ChatGPT process is running in this test; closing must not touch anything real.
function Get-Process { param($Id,$Name,$ErrorAction); if ($null -ne $Id) { return [pscustomobject]@{SessionId=1} }; return @() }
function Start-Process { throw 'Must not start an executable' }
function Stop-Process { throw 'Must not close an App' }
function Find-DesktopBackend { throw 'Must not locate a backend' }
function Invoke-CodexExport { throw 'Must not run a backend' }
$config="model=`"gpt-existing`"`r`n[features]`r`ngoals=true`r`n"
[IO.File]::WriteAllText((Join-Path $target 'config.toml'),$config,(New-Object Text.UTF8Encoding($false)))

# Exercise the exact public HTTP bytes and the documented PowerShell pipeline.
$Error.Clear()
irm $SourceUrl | iex
if (@($Error | Where-Object {$_.Exception -is [System.Management.Automation.CommandNotFoundException] -or $_.CategoryInfo.Category -eq 'ParserError'}).Count) { throw 'HTTP pipeline produced PowerShell parsing errors' }
$catalog=ConvertFrom-Json -InputObject ([IO.File]::ReadAllText((Join-Path $target 'myautoapi-models.json')))
if (@($catalog.models | Where-Object {$_.slug -ceq 'minimax-m3'}).Count -ne 1) { throw 'Expected exactly one MiniMax M3' }
if (@($catalog.models | Where-Object {$_.slug -cmatch '^gpt-'}).Count -eq 0) { throw 'GPT catalog missing' }
$written=[IO.File]::ReadAllText((Join-Path $target 'config.toml'))
if (-not $written.StartsWith("model=`"gpt-existing`"`r`n") -or -not $written.Contains("[features]`r`ngoals=true`r`n")) { throw 'Default model or other settings changed' }
if (@($catalog.models | Where-Object {$_.slug -ceq 'codex-auto-review'}).Count -ne 1) { throw 'Hidden official model missing' }
if ((Get-ChildItem -LiteralPath $target -Recurse -File -Filter '*.exe').Count) { throw 'Unexpected executable created' }
# A present CLI must have no influence either. Any invocation would fail this run.
function codex { throw 'Must not run an installed CLI' }
$before=[IO.File]::ReadAllText((Join-Path $target 'config.toml'))
irm $SourceUrl | iex
if ([IO.File]::ReadAllText((Join-Path $target 'config.toml')) -cne $before) { throw 'CLI presence changed the generated config' }
Start-ModelInstaller '' '' -Uninstall
$remaining=ConvertFrom-Json -InputObject ([IO.File]::ReadAllText((Join-Path $target 'myautoapi-models.json')))
if (@($remaining.models | Where-Object {$_.slug -ceq 'minimax-m3'}).Count) { throw 'Uninstall did not remove managed model' }
Write-Output 'PowerShell HTTP irm | iex import/update/uninstall passed with and without a CLI; no executable/backend or process control needed'
