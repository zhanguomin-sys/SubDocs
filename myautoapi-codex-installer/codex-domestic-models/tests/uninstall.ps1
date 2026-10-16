param([string]$Backend, [string]$WorkRoot)
$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot '../src/core.ps1')
. (Join-Path $PSScriptRoot '../src/windows-host.ps1')
$script:assertions=0
function Assert($Value) { if (-not $Value) { throw "Assertion $($script:assertions+1) failed at line $((Get-PSCallStack)[1].ScriptLineNumber)" }; $script:assertions++ }
$null=[IO.Directory]::CreateDirectory($WorkRoot)
$fixture=ConvertFrom-Json -InputObject (Invoke-CodexExport $Backend @('debug','models','--bundled') $WorkRoot)
$gpt=@($fixture.models | Where-Object {$_.slug -like 'gpt-*'})[0]
$gpt | Add-Member custom_metadata ([pscustomobject]@{keep=@('one','two')})
function Copy-Model($Model,[string]$Slug) {
  $copy=ConvertFrom-Json -InputObject (ConvertTo-Json -InputObject $Model -Depth 100)
  $copy.slug=$Slug
  return $copy
}
function Setup([string]$Name) {
  $target=Join-Path $WorkRoot ($Name+'-'+[Guid]::NewGuid().ToString('N'))
  $null=[IO.Directory]::CreateDirectory($target)
  $catalog=[pscustomobject]@{models=@($gpt,(Copy-Model $gpt 'managed-one'),(Copy-Model $gpt 'managed-two'),(Copy-Model $gpt 'Managed-one'),(Copy-Model $gpt 'user-custom')); unknown=[pscustomobject]@{preserve=@('top','level')}}
  Write-PrivateText (Join-Path $target 'myautoapi-models.json') (ConvertTo-Json -InputObject $catalog -Depth 100)
  Write-PrivateText (Join-Path $target 'myautoapi-models.managed.json') '["managed-one","managed-two","previous-release-model"]'
  $config='# original comments'+"`r`n"+'model_catalog_json = '+(ConvertTo-Json -InputObject (Join-Path $target 'myautoapi-models.json') -Compress)+"`r`n"+'model = "managed-one"'+"`r`n"+'model_provider = "customer"'+"`r`n"+'[desktop]'+"`r`n"+'theme = "dark"'+"`r`n"+'enabled-reasoning-efforts = ["high", "max"]'+"`r`n"
  [IO.File]::WriteAllText((Join-Path $target 'config.toml'),$config,(New-Object Text.UTF8Encoding($true)))
  Write-PrivateText (Join-Path $target 'auth.json') 'sentinel-do-not-read'
  return $target
}
function Get-State([string]$Target) {
  return ,@(@('config.toml','myautoapi-models.json','myautoapi-models.managed.json','auth.json') | ForEach-Object {
    $file=Join-Path $Target $_
    if ([IO.File]::Exists($file)) { [Convert]::ToBase64String([IO.File]::ReadAllBytes($file)) } else { $null }
  })
}
function Assert-State($Actual,$Expected) {
  for ($index=0; $index -lt $Actual.Count; $index++) { Assert (($null -eq $Actual[$index]) -eq ($null -eq $Expected[$index])); Assert ($Actual[$index] -ceq $Expected[$index]) }
}
$target=Setup 'active'
$before=Get-State $target
$beforeCatalog=ConvertFrom-Json -InputObject (Read-PrivateText (Join-Path $target 'myautoapi-models.json'))
$result=Remove-ImportedModels $target
Assert ($result.Changed -and $result.Active -and $result.Count -eq 2)
$after=Get-State $target
Assert ($after[0] -ceq $before[0]); Assert ($after[3] -ceq $before[3])
Assert (@(Get-ChildItem -LiteralPath $target -Filter 'config.toml.bak.*').Count -eq 0)
$catalog=ConvertFrom-Json -InputObject (Read-PrivateText (Join-Path $target 'myautoapi-models.json'))
Assert ($catalog.models.Count -eq 3)
Assert (@($catalog.models | Where-Object {$_.slug -ceq 'Managed-one'}).Count -eq 1)
Assert ((ConvertTo-Json -InputObject $catalog.unknown -Compress) -ceq (ConvertTo-Json -InputObject $beforeCatalog.unknown -Compress))
foreach ($model in $catalog.models) {
  $original=@($beforeCatalog.models | Where-Object {$_.slug -ceq $model.slug})[0]
  Assert ((ConvertTo-Json -InputObject $model -Depth 100 -Compress) -ceq (ConvertTo-Json -InputObject $original -Depth 100 -Compress))
}
Assert ((Read-PrivateText (Join-Path $target 'myautoapi-models.managed.json')).Trim() -ceq '[]')
$backupCount=@(Get-ChildItem -LiteralPath $target -Filter '*.bak.*').Count
$result=Remove-ImportedModels $target
Assert (-not $result.Changed -and -not $result.Active -and $result.Count -eq 0)
Assert-State (Get-State $target) $after
Assert (@(Get-ChildItem -LiteralPath $target -Filter '*.bak.*').Count -eq $backupCount)

# The managed record alone determines whether there is anything to uninstall.
$absent=Join-Path $WorkRoot ('absent-'+[Guid]::NewGuid().ToString('N'))
Assert (-not (Remove-ImportedModels $absent).Changed); Assert (-not (Test-PrivatePath $absent))
foreach ($mode in @('missing','empty')) {
  $target=Setup ('noop-'+$mode)
  if ($mode -ceq 'missing') { [IO.File]::Delete((Join-Path $target 'myautoapi-models.managed.json')) }
  else { Write-PrivateText (Join-Path $target 'myautoapi-models.managed.json') '[]' }
  Write-PrivateText (Join-Path $target 'config.toml') 'invalid TOML private-credential-do-not-print'
  Write-PrivateText (Join-Path $target 'myautoapi-models.json') 'invalid catalog'
  $before=Get-State $target
  Assert (-not (Remove-ImportedModels $target).Changed)
  Assert-State (Get-State $target) $before
  Assert (@(Get-ChildItem -LiteralPath $target -Filter '*.bak.*').Count -eq 0)
}

# Other active catalog files are neither read nor changed; missing config stays absent.
foreach ($mode in @('external','relative','missing','empty','no-pointer','zero-matches','inline-desktop','profile-catalog')) {
  $target=Setup $mode
  $configPath=Join-Path $target 'config.toml'
  $external=Join-Path $target 'external-models.json'
  Write-PrivateText $external 'private external catalog is not parsed'
  if ($mode -ceq 'external') { Write-PrivateText $configPath ('model_catalog_json = '+(ConvertTo-Json -InputObject $external -Compress)) }
  elseif ($mode -ceq 'relative') { Write-PrivateText $configPath "model_catalog_json = 'myautoapi-models.json'`n" }
  elseif ($mode -ceq 'missing') { [IO.File]::Delete($configPath) }
  elseif ($mode -ceq 'empty') { Write-PrivateText $configPath '' }
  elseif ($mode -ceq 'no-pointer') { Write-PrivateText $configPath "model = 'customer-default'`n" }
  elseif ($mode -ceq 'zero-matches') { Write-PrivateText (Join-Path $target 'myautoapi-models.managed.json') '["already-removed"]' }
  elseif ($mode -ceq 'inline-desktop') { Write-PrivateText $configPath ("model_catalog_json = 'myautoapi-models.json'`ndesktop = { theme = 'dark', enabled-reasoning-efforts = ['unrelated-effort'] }`n") }
  elseif ($mode -ceq 'profile-catalog') { Write-PrivateText $configPath ("model_catalog_json = 'myautoapi-models.json'`n[profiles.customer]`nmodel_catalog_json = 'external-models.json'`n") }
  $before=Get-State $target
  $result=Remove-ImportedModels $target
  Assert ($result.Changed); Assert ($result.Active -eq ($mode -in @('relative','inline-desktop')))
  Assert ($result.Count -eq $(if ($mode -ceq 'zero-matches') { 0 } else { 2 }))
  $after=Get-State $target
  Assert ($after[0] -ceq $before[0]); Assert (($null -eq $after[0]) -eq ($null -eq $before[0]))
  Assert ($after[3] -ceq $before[3])
  Assert ((Read-PrivateText $external) -ceq 'private external catalog is not parsed')
  Assert (@(Get-ChildItem -LiteralPath $target -Filter 'config.toml.bak.*').Count -eq 0)
}

foreach ($bad in @('{}','null','[null]','[1]','[[]]','[""]','["gpt-do-not-remove"]','["GPT-do-not-remove"]','["unterminated]')) {
  $target=Setup 'bad-managed'
  Write-PrivateText (Join-Path $target 'myautoapi-models.managed.json') $bad
  $before=Get-State $target; $failed=$false; $diagnostic=''
  try { $null=Remove-ImportedModels $target } catch { $failed=$true; $diagnostic=Format-ModelImportError $_ -Uninstall }
  Assert $failed; Assert-State (Get-State $target) $before
  Assert ($diagnostic.Contains((Join-Path $target 'myautoapi-models.managed.json')))
  Assert ($diagnostic.Contains('解析已管理模型记录') -or $diagnostic.Contains('校验已管理模型记录'))
  Assert (@(Get-ChildItem -LiteralPath $target -Filter '*.bak.*').Count -eq 0)
}
foreach ($bad in @($null,'invalid JSON','{"models":[]}')) {
  $target=Setup 'bad-catalog'; $catalogPath=Join-Path $target 'myautoapi-models.json'
  if ($null -eq $bad) { [IO.File]::Delete($catalogPath) } else { Write-PrivateText $catalogPath $bad }
  $before=Get-State $target; $failed=$false; $diagnostic=''
  try { $null=Remove-ImportedModels $target } catch { $failed=$true; $diagnostic=Format-ModelImportError $_ -Uninstall }
  Assert $failed; Assert-State (Get-State $target) $before
  Assert ($diagnostic.Contains($catalogPath))
}

$script:realMove=${function:Move-StagedFile}
$script:realRead=${function:Read-PrivateText}
$script:realWrite=${function:Write-PrivateText}
foreach ($fault in @('write','final-validation','before-commit','after-commit','delete-config')) {
  $target=Setup ('failure-'+$fault)
  $before=Get-State $target
  $script:target=$target; $script:fault=$fault; $script:faulted=$false; $script:finalValidation=$false
  function Move-StagedFile([string]$From,[string]$To) {
    $managed=$To -ceq (Join-Path $script:target 'myautoapi-models.managed.json')
    if ($managed -and -not $script:faulted -and $script:fault -ceq 'write') { $script:faulted=$true; throw 'injected write failure' }
    & $script:realMove $From $To
    if ($managed -and -not $script:faulted) {
      if ($script:fault -ceq 'final-validation') { $script:finalValidation=$true; $script:faulted=$true }
      elseif ($script:fault -ceq 'after-commit') { [IO.File]::AppendAllText((Join-Path $script:target 'config.toml'),"# external edit`n"); $script:faulted=$true }
      elseif ($script:fault -ceq 'delete-config') { [IO.File]::Delete((Join-Path $script:target 'config.toml')); $script:faulted=$true }
    }
  }
  function Read-PrivateText([string]$Path) {
    if ($script:finalValidation -and $Path -ceq (Join-Path $script:target 'myautoapi-models.json')) {
      $script:finalValidation=$false
      throw (New-ModelImportError 'injected final validation')
    }
    return (& $script:realRead $Path)
  }
  function Write-PrivateText([string]$Path,[string]$Text) {
    & $script:realWrite $Path $Text
    if ($script:fault -ceq 'before-commit' -and -not $script:faulted -and $Path.EndsWith('file-0')) {
      [IO.File]::AppendAllText((Join-Path $script:target 'config.toml'),"# external edit`n"); $script:faulted=$true
    }
  }
  $failed=$false; $diagnostic=''
  try { $null=Remove-ImportedModels $target } catch { $failed=$true; $diagnostic=Format-ModelImportError $_ -Uninstall }
  Set-Item Function:Move-StagedFile $script:realMove
  Set-Item Function:Read-PrivateText $script:realRead
  Set-Item Function:Write-PrivateText $script:realWrite
  Assert $failed; Assert $script:faulted; Assert ($diagnostic.StartsWith('卸载失败'))
  $after=Get-State $target
  if ($fault -in @('write','final-validation')) { Assert-State $after $before }
  else {
    if ($fault -ceq 'delete-config') { Assert ($null -eq $after[0]) }
    else { Assert ((Read-PrivateText (Join-Path $target 'config.toml')).EndsWith("# external edit`n")) }
    Assert ($after[3] -ceq $before[3])
    if ($fault -ceq 'before-commit') { Assert ($after[1] -ceq $before[1]); Assert ($after[2] -ceq $before[2]) }
    else {
      Assert ($diagnostic.Contains('已保留当前内容和备份'))
      Assert ((Read-PrivateText (Join-Path $target 'myautoapi-models.managed.json')).Trim() -ceq '[]')
    }
  }
  Assert (@(Get-ChildItem -LiteralPath $target -Filter 'config.toml.bak.*').Count -eq 0)
  Assert (@(Get-ChildItem -LiteralPath $target -Directory -Force | Where-Object {$_.Name.StartsWith('.myautoapi-models-')}).Count -eq 0)
}

# The uninstall entry does not need a backend and closes clients only for active changes.
$script:closed=0
function Find-DesktopBackend([string]$ExplicitPath) { throw 'Uninstall must not locate the backend' }
function Invoke-CodexExport([string]$Exe,[string[]]$Arguments,[string]$TargetHome) { throw 'Uninstall must not call a backend' }
function Stop-RunningClients { $script:closed++; return @() }
function Report-RunningClients($Clients) { }
$oldOs=$env:OS
try {
  $env:OS='Windows_NT'
  $target=Setup 'entry-active'
  Start-ModelInstaller $target '' -Uninstall
  Assert ($script:closed -eq 1)
  Start-ModelInstaller $target '' -Uninstall
  Assert ($script:closed -eq 1)
  $target=Setup 'entry-inactive'
  Write-PrivateText (Join-Path $target 'config.toml') "model_catalog_json = 'external.json'`n"
  Start-ModelInstaller $target '' -Uninstall
  Assert ($script:closed -eq 1)
} finally { $env:OS=$oldOs }
Write-Output "PowerShell uninstall: $script:assertions assertions passed; no real process was stopped"
