# Uninstall removes only this tool's models and never touches config.toml.
param([string]$DomesticPath, [string]$WorkRoot)
$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot '../src/core.ps1')
. (Join-Path $PSScriptRoot '../src/windows-host.ps1')
$DomesticCatalog=ConvertFrom-Json -InputObject ([IO.File]::ReadAllText($DomesticPath))
$BundledOfficialCatalog=ConvertFrom-Json -InputObject ([IO.File]::ReadAllText((Join-Path $PSScriptRoot '../src/gpt-catalog.json')))
$BundledOfficialVersion='0.162.1'
$script:assertions=0
function Assert($Value) { if (-not $Value) { throw "Assertion $($script:assertions+1) failed at line $((Get-PSCallStack)[1].ScriptLineNumber)" }; $script:assertions++ }
$target=Join-Path $WorkRoot ('uninstall-'+[Guid]::NewGuid().ToString('N'))
$null=[IO.Directory]::CreateDirectory($target)
[IO.File]::WriteAllText((Join-Path $target 'config.toml'),"model = `"gpt-5.5`"`r`n# keep`r`n")
[IO.File]::WriteAllText((Join-Path $target 'auth.json'),'sentinel-do-not-read')
$null=Invoke-ModelImport $target $DomesticCatalog
$catalogPath=Join-Path $target 'myautoapi-models.json'
$catalog=ConvertFrom-Json -InputObject (Read-PrivateText $catalogPath)
$custom=ConvertFrom-Json -InputObject (ConvertTo-Json -InputObject $catalog.models[0] -Depth 100)
$custom.slug='user-custom'
$catalog | Add-Member -NotePropertyName models -NotePropertyValue (@($catalog.models)+$custom) -Force
Write-PrivateText $catalogPath (ConvertTo-Json -InputObject $catalog -Depth 100)
Write-PrivateText (Join-Path $target 'myautoapi-models.managed.json') '["removed-in-new-release"]'
$config=[IO.File]::ReadAllBytes((Join-Path $target 'config.toml'))
$ids=@($catalog.models | ForEach-Object {$_.slug})
$ids2=@($ids + 'removed-in-new-release')
$result=Remove-ImportedModels $target $DomesticCatalog
Assert ($result.Changed -and $result.Active -and $result.Count -eq $DomesticCatalog.models.Count)
$after=ConvertFrom-Json -InputObject (Read-PrivateText $catalogPath)
$left=@($after.models | ForEach-Object {$_.slug})
Assert ($left -contains 'user-custom' -and $left -contains 'codex-auto-review' -and $left -contains 'gpt-6-astra')
foreach ($model in $DomesticCatalog.models) { Assert ($left -notcontains $model.slug) }
Assert ([Convert]::ToBase64String([IO.File]::ReadAllBytes((Join-Path $target 'config.toml'))) -ceq [Convert]::ToBase64String($config))
Assert ([IO.File]::ReadAllText((Join-Path $target 'auth.json')) -ceq 'sentinel-do-not-read')
Assert ((Read-PrivateText (Join-Path $target 'myautoapi-models.managed.json')).Trim() -ceq '[]')
$again=Remove-ImportedModels $target $DomesticCatalog
Assert (-not $again.Changed)
$empty=Join-Path $WorkRoot ('uninstall-empty-'+[Guid]::NewGuid().ToString('N'))
$null=[IO.Directory]::CreateDirectory($empty)
Assert (-not (Remove-ImportedModels $empty $DomesticCatalog).Changed)
# A catalog that config.toml does not point at is cleaned without reporting it as in use.
$other=Join-Path $WorkRoot ('uninstall-inactive-'+[Guid]::NewGuid().ToString('N'))
$null=Invoke-ModelImport $target $DomesticCatalog
$null=[IO.Directory]::CreateDirectory($other)
Copy-Item (Join-Path $target 'myautoapi-models.json') $other
[IO.File]::WriteAllText((Join-Path $other 'config.toml'),"model = `"x`"`
")
$inactive=Remove-ImportedModels $other $DomesticCatalog
Assert ($inactive.Changed -and -not $inactive.Active)
Write-Output "PowerShell uninstall checks passed ($($script:assertions) assertions)"
