# File-level behaviour of the Windows importer: backups, temp files, BOM, concurrent edits.
param([string]$DomesticPath, [string]$WorkRoot)
$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot '../src/core.ps1')
. (Join-Path $PSScriptRoot '../src/windows-host.ps1')
$DomesticCatalog=ConvertFrom-Json -InputObject ([IO.File]::ReadAllText($DomesticPath))
$BundledOfficialCatalog=ConvertFrom-Json -InputObject ([IO.File]::ReadAllText((Join-Path $PSScriptRoot '../src/gpt-catalog.json')))
$BundledOfficialVersion='0.162.1'
$script:assertions=0
function Assert($Value) { if (-not $Value) { throw "Assertion $($script:assertions+1) failed at line $((Get-PSCallStack)[1].ScriptLineNumber)" }; $script:assertions++ }
function New-Home([string]$Name) {
  $target=Join-Path $WorkRoot ($Name+'-'+[Guid]::NewGuid().ToString('N'))
  $null=[IO.Directory]::CreateDirectory($target)
  return $target
}

# A fresh directory gets exactly the three files, no temporary leftovers.
$target=New-Home 'fresh'
$result=Invoke-ModelImport $target $DomesticCatalog
Assert ($result.Count -eq $DomesticCatalog.models.Count)
Assert (-not $result.UsedCache)
Assert ((@(Get-ChildItem -LiteralPath $target -Force).Name | Sort-Object) -join ',' -ceq 'config.toml,myautoapi-models.json,myautoapi-models.managed.json')
$catalog=ConvertFrom-Json -InputObject (Read-PrivateText (Join-Path $target 'myautoapi-models.json'))
$auto=@($catalog.models | Where-Object {$_.slug -ceq 'codex-auto-review'})
Assert ($auto.Count -eq 1 -and $auto[0].visibility -ceq 'hide')

# Re-import is stable, backs up the previous files and leaves no temporary files.
$first=@{}
foreach ($name in @('config.toml','myautoapi-models.json','myautoapi-models.managed.json')) { $first[$name]=[Convert]::ToBase64String([IO.File]::ReadAllBytes((Join-Path $target $name))) }
$null=Invoke-ModelImport $target $DomesticCatalog
foreach ($name in @('config.toml','myautoapi-models.json','myautoapi-models.managed.json')) {
  Assert ([Convert]::ToBase64String([IO.File]::ReadAllBytes((Join-Path $target $name))) -ceq $first[$name])
  Assert (@(Get-ChildItem -LiteralPath $target -Filter ($name+'.bak.*')).Count -eq 1)
}
Assert (@(Get-ChildItem -LiteralPath $target -Filter '*.tmp.*').Count -eq 0)

# A leading BOM stays in front, once.
$target=New-Home 'bom'
[IO.File]::WriteAllText((Join-Path $target 'config.toml'),"model = `"x`"`r`n[desktop]`r`ntheme = `"dark`"`r`n",(New-Object Text.UTF8Encoding($true)))
$null=Invoke-ModelImport $target $DomesticCatalog
$bytes=[IO.File]::ReadAllBytes((Join-Path $target 'config.toml'))
Assert ($bytes[0] -eq 0xEF -and $bytes[1] -eq 0xBB -and $bytes[2] -eq 0xBF -and -not ($bytes[3] -eq 0xEF -and $bytes[4] -eq 0xBB))

# Bad config: nothing is written or backed up.
$target=New-Home 'badconfig'
[IO.File]::WriteAllText((Join-Path $target 'config.toml'),"a = `"unterminated`n")
$failed=$false
try { $null=Invoke-ModelImport $target $DomesticCatalog } catch { $failed=$true }
Assert $failed
Assert ((@(Get-ChildItem -LiteralPath $target -Force).Name) -join ',' -ceq 'config.toml')

# A config edited while the import was running is not overwritten.
$target=New-Home 'race'
$config=Join-Path $target 'config.toml'
[IO.File]::WriteAllText($config,"model = `"x`"`n")
$files=@(
  @{Path=(Join-Path $target 'myautoapi-models.json'); Text='{"models":[{"slug":"a"}]}'; Kind='catalog'},
  @{Path=$config; Text="model = `"y`"`n"; Kind='config'}
)
[IO.File]::WriteAllText($config,"model = `"changed`"`n")
$failed=$false
try { Write-ModelFiles $files "model = `"x`"`n" } catch { $failed=$true }
Assert $failed
Assert ([IO.File]::ReadAllText($config) -ceq "model = `"changed`"`n")
Assert (-not [IO.File]::Exists((Join-Path $target 'myautoapi-models.json')))
Assert (@(Get-ChildItem -LiteralPath $target -Filter '*.tmp.*').Count -eq 0)
Write-Output "PowerShell file import checks passed ($($script:assertions) assertions)"
