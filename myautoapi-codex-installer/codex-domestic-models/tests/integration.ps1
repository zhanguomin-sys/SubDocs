param([string]$Backend, [string]$DomesticPath, [string]$WorkRoot)
$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot '../src/core.ps1')
. (Join-Path $PSScriptRoot '../src/windows-host.ps1')
$Domestic=ConvertFrom-Json -InputObject ([IO.File]::ReadAllText($DomesticPath))
$script:assertions=0
function Assert($Value) { if (-not $Value) { throw 'Assertion failed' }; $script:assertions++ }
function Setup([string]$Name) {
  $homePath=Join-Path $WorkRoot ($Name+'-'+[Guid]::NewGuid().ToString('N'))
  $null=[IO.Directory]::CreateDirectory($homePath)
  $exported=ConvertFrom-Json -InputObject (Invoke-CodexExport $Backend @('debug','models','--bundled') $homePath)
  $selected=@($exported.models | Where-Object {$_.slug -like 'gpt-*'})[0].slug
  $config=@"
# preserve comments
model = "$selected"
review_model = "$selected"
model_provider = "relay"
model_reasoning_effort = "high"
[model_providers.relay]
name = "relay"
base_url = "https://example.invalid/v1"
wire_api = "responses"
requires_openai_auth = false
[desktop]
theme = "dark"
"@
  Write-PrivateText (Join-Path $homePath 'config.toml') ($config+"`n")
  Write-PrivateText (Join-Path $homePath 'auth.json') 'sentinel-do-not-read'
  return $homePath
}
$null=[IO.Directory]::CreateDirectory($WorkRoot)
$target=Setup 'success'
Assert ((Invoke-ModelImport $target $Backend $Domestic) -eq @($Domestic.models).Count)
$catalogPath=Join-Path $target 'myautoapi-models.json'
$value=ConvertFrom-Json -InputObject (Read-PrivateText $catalogPath)
Assert (@($value.models | Where-Object {$_.slug -like 'gpt-*'}).Count -gt 0)
Assert ((Read-PrivateText (Join-Path $target 'auth.json')) -ceq 'sentinel-do-not-read')
Assert (@(Get-ChildItem -LiteralPath $target -Filter 'config.toml.bak.*').Count -eq 1)
$gpt=@($value.models | Where-Object {$_.slug -like 'gpt-*'})[0]
$gpt | Add-Member custom_metadata @{keep=$true}
$owned=@($value.models | Where-Object {$_.slug -ceq $Domestic.models[0].slug})[0]
$owned | Add-Member obsolete $true
$user=ConvertFrom-Json -InputObject (ConvertTo-Json -InputObject $gpt -Depth 100); $user.slug='user-custom'
$removed=ConvertFrom-Json -InputObject (ConvertTo-Json -InputObject $gpt -Depth 100); $removed.slug='removed-domestic'
$value.models+=@($user,$removed)
Write-PrivateText $catalogPath (ConvertTo-Json -InputObject $value -Depth 100)
Write-PrivateText (Join-Path $target 'myautoapi-models.managed.json') (ConvertTo-Json -InputObject (@($Domestic.models | ForEach-Object {$_.slug})+@('removed-domestic')))
Assert ((Invoke-ModelImport $target $Backend $Domestic) -eq @($Domestic.models).Count)
$value=ConvertFrom-Json -InputObject (Read-PrivateText $catalogPath)
Assert (@($value.models | Where-Object {$_.slug -ceq 'removed-domestic'}).Count -eq 0)
Assert (@($value.models | Where-Object {$_.slug -ceq 'user-custom'})[0].custom_metadata.keep)
Assert (@($value.models | Where-Object {$_.slug -ceq $gpt.slug})[0].custom_metadata.keep)
Assert ($null -eq @($value.models | Where-Object {$_.slug -ceq $Domestic.models[0].slug})[0].obsolete)
$beforeConfig=Read-PrivateText (Join-Path $target 'config.toml')
$null=Invoke-ModelImport $target $Backend $Domestic
Assert ((Read-PrivateText (Join-Path $target 'config.toml')) -ceq $beforeConfig)

$script:realExport=${function:Invoke-CodexExport}
$script:realMove=${function:Move-StagedFile}
$script:realRead=${function:Read-PrivateText}
function Get-CurrentTransactionFiles([string]$Root) {
  return ,@(@('config.toml','myautoapi-models.json','myautoapi-models.managed.json') | ForEach-Object {
    $file=Join-Path $Root $_
    if ([IO.File]::Exists($file)) { [Convert]::ToBase64String([IO.File]::ReadAllBytes($file)) } else { '' }
  })
}
foreach ($failure in @('write','validation','concurrent','wrong_catalog','postcommit','wrong_metadata','changed_config','changed_managed','changed_catalog','changed_catalog_backend_error','deleted_config','unreadable_catalog','write_and_external_edit')) {
  foreach ($existing in @($false,$true)) {
    $target=Setup $failure
    if ($existing) { $null=Invoke-ModelImport $target $Backend $Domestic }
    $paths=@('config.toml','myautoapi-models.json','myautoapi-models.managed.json')
    $before=@($paths | ForEach-Object { $p=Join-Path $target $_; if ([IO.File]::Exists($p)) { [Convert]::ToBase64String([IO.File]::ReadAllBytes($p)) } else { '' } })
    $script:faulted=$false; $script:failure=$failure; $script:target=$target; $script:preserved=$null; $script:blockedRead=$false
    function Invoke-CodexExport([string]$Exe,[string[]]$Arguments,[string]$TargetHome) {
      $final=$false
      if ($TargetHome -cne $script:target) {
        $validationInfo=Get-ConfigInfo (Read-PrivateText (Join-Path $TargetHome 'config.toml'))
        $final=(Get-TomlString $validationInfo.Values['model_catalog_json']) -ceq (Join-Path $script:target 'myautoapi-models.json')
        if ($script:failure -ceq 'validation') { throw 'injected validation' }
        if ($script:failure -ceq 'concurrent') { [IO.File]::AppendAllText((Join-Path $script:target 'config.toml'),"# edited externally`n") }
        if ($script:failure -ceq 'wrong_catalog') { return (ConvertTo-Json -InputObject $Domestic -Depth 100) }
      }
      if ($final -and $script:failure -ceq 'postcommit') { throw 'injected final validation' }
      if ($final -and $script:failure -in @('changed_config','changed_managed','changed_catalog')) {
        $names=@{changed_config='config.toml'; changed_managed='myautoapi-models.managed.json'; changed_catalog='myautoapi-models.json'}
        [IO.File]::AppendAllText((Join-Path $script:target $names[$script:failure]),"`n")
        $script:preserved=Get-CurrentTransactionFiles $script:target
      }
      if ($final -and $script:failure -ceq 'changed_catalog_backend_error') {
        [IO.File]::AppendAllText((Join-Path $script:target 'myautoapi-models.json'),"`n")
        $script:preserved=Get-CurrentTransactionFiles $script:target
        throw 'injected final backend error after external edit'
      }
      if ($final -and $script:failure -ceq 'deleted_config') {
        [IO.File]::Delete((Join-Path $script:target 'config.toml'))
        $script:preserved=Get-CurrentTransactionFiles $script:target
      }
      if ($final -and $script:failure -ceq 'unreadable_catalog') {
        $script:preserved=Get-CurrentTransactionFiles $script:target
        $script:blockedRead=$true
      }
      $text=& $script:realExport $Exe $Arguments $TargetHome
      if ($final -and $script:failure -ceq 'wrong_metadata') {
        $data=ConvertFrom-Json -InputObject $text
        @($data.models | Where-Object {$_.slug -ceq $Domestic.models[0].slug})[0].context_window=42
        return (ConvertTo-Json -InputObject $data -Depth 100)
      }
      return $text
    }
    function Move-StagedFile([string]$From,[string]$To) {
      if ($script:failure -ceq 'write' -and -not $script:faulted -and $To.EndsWith('myautoapi-models.managed.json')) { $script:faulted=$true; throw 'injected write' }
      if ($script:failure -ceq 'write_and_external_edit' -and -not $script:faulted -and $To.EndsWith('myautoapi-models.managed.json')) {
        $script:faulted=$true
        [IO.File]::AppendAllText((Join-Path $script:target 'config.toml'),"# saved by App before closure`n")
        $script:preserved=Get-CurrentTransactionFiles $script:target
        throw 'injected write failure after external edit'
      }
      & $script:realMove $From $To
    }
    function Read-PrivateText([string]$Path) {
      if ($script:blockedRead -and $Path -ceq (Join-Path $script:target 'myautoapi-models.json')) { throw (New-Object IO.IOException -ArgumentList 'private-credential-do-not-print') }
      return (& $script:realRead $Path)
    }
    $failed=$false; $diagnostic=''
    try { $null=Invoke-ModelImport $target $Backend $Domestic } catch { $failed=$true; $diagnostic=Format-ModelImportError $_ }
    Set-Item Function:Read-PrivateText $script:realRead
    Assert $failed
    for ($index=0;$index -lt $paths.Count;$index++) {
      $file=Join-Path $target $paths[$index]
      $actual=if ([IO.File]::Exists($file)) { [Convert]::ToBase64String([IO.File]::ReadAllBytes($file)) } else { '' }
      $expected=$before[$index]
      if ($null -ne $script:preserved) { $expected=$script:preserved[$index] }
      if ($failure -ceq 'concurrent' -and $index -eq 0) {
        $text=[Text.Encoding]::UTF8.GetString([Convert]::FromBase64String($expected))+"# edited externally`n"
        $expected=[Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($text))
      }
      Assert ($actual -ceq $expected)
    }
    if ($null -ne $script:preserved) {
      Assert ($diagnostic.Contains('导入过程中出现外部修改，已保留当前内容和备份，软件未关闭。'))
      $conflict=if ($failure -in @('changed_config','deleted_config','write_and_external_edit')) { 'config.toml' } elseif ($failure -ceq 'changed_managed') { 'myautoapi-models.managed.json' } else { 'myautoapi-models.json' }
      Assert ($diagnostic.Contains((Join-Path $target $conflict)))
      Assert (@(Get-ChildItem -LiteralPath $target -Filter 'config.toml.bak.*').Count -gt 0)
      Assert (-not $diagnostic.Contains('private-credential-do-not-print'))
      if ($failure -ceq 'deleted_config') { Assert (-not (Test-PrivatePath (Join-Path $target 'config.toml'))) }
    }
    Assert ((Read-PrivateText (Join-Path $target 'auth.json')) -ceq 'sentinel-do-not-read')
    Assert (@(Get-ChildItem -LiteralPath $target -Directory -Force | Where-Object {$_.Name.StartsWith('.myautoapi-models-')}).Count -eq 0)
    Set-Item Function:Invoke-CodexExport $script:realExport
    Set-Item Function:Move-StagedFile $script:realMove
  }
}
foreach ($invalid in @('directory','symlink','dangling','managed_directory')) {
  $target=Setup $invalid
  $filename=if ($invalid -ceq 'managed_directory') { 'myautoapi-models.managed.json' } else { 'myautoapi-models.json' }
  $destination=Join-Path $target $filename
  $linkTarget=Join-Path $target 'link-target'
  if ($invalid -ceq 'directory' -or $invalid -ceq 'managed_directory') { $null=[IO.Directory]::CreateDirectory($destination) }
  else {
    if ($invalid -ceq 'symlink') { Write-PrivateText $linkTarget 'untouched' }
    $null=New-Item -ItemType SymbolicLink -Path $destination -Target $linkTarget
  }
  $before=Read-PrivateText (Join-Path $target 'config.toml')
  $failed=$false; try { $null=Invoke-ModelImport $target $Backend $Domestic } catch { $failed=$true }
  Assert $failed
  Assert ((Read-PrivateText (Join-Path $target 'config.toml')) -ceq $before)
  Assert (Test-PrivatePath $destination)
  if ($invalid -ceq 'symlink') { Assert ((Read-PrivateText $linkTarget) -ceq 'untouched') }
  if ($invalid -ceq 'dangling') { Assert (-not [IO.File]::Exists($linkTarget)) }
}
$target=Setup 'missing_source'
$configPath=Join-Path $target 'config.toml'
Write-PrivateText $configPath ('model_catalog_json="missing.json"'+"`n"+(Read-PrivateText $configPath))
Assert ((Invoke-ModelImport $target $Backend $Domestic) -eq @($Domestic.models).Count)
$target=Setup 'relative_instructions'
$configPath=Join-Path $target 'config.toml'
Write-PrivateText (Join-Path $target 'instructions.md') "Test repository instructions.`n"
Write-PrivateText $configPath ('model_instructions_file="instructions.md"'+"`n"+(Read-PrivateText $configPath))
Assert ((Invoke-ModelImport $target $Backend $Domestic) -eq @($Domestic.models).Count)
Assert ((Read-PrivateText $configPath).Contains('model_instructions_file="instructions.md"'))

# Official accounts and arbitrary service names/URLs do not depend on a relay installer.
foreach ($text in @('', 'model_provider="OpenAI"', "model_provider=`"customer`"`n[model_providers.customer]`nname=`"customer`"`nbase_url=`"https://customer.example.invalid/custom/v1`"`nwire_api=`"responses`"`nrequires_openai_auth=false`nexperimental_bearer_token=`"private-credential-do-not-print`"")) {
  $target=Setup 'provider-independent'
  $configPath=Join-Path $target 'config.toml'
  Write-PrivateText $configPath $text
  Assert ((Invoke-ModelImport $target $Backend $Domestic) -eq @($Domestic.models).Count)
  $after=Read-PrivateText $configPath
  $restored=[regex]::Replace($after,'(?m)^model_catalog_json = [^\r\n]*\r?\n','')
  $restored=[regex]::Replace($restored,'\r?\n\[desktop\]\r?\nenabled-reasoning-efforts = [^\r\n]*\r?\n$','')
  Assert ($restored.TrimEnd("`r","`n") -ceq $text.TrimEnd("`r","`n"))
  Assert ((Read-PrivateText (Join-Path $target 'auth.json')) -ceq 'sentinel-do-not-read')
}
foreach ($mode in @('missing','empty','new_home')) {
  $target=Setup ('initial-'+$mode)
  if ($mode -ceq 'missing') { [IO.File]::Delete((Join-Path $target 'config.toml')) }
  elseif ($mode -ceq 'empty') { Write-PrivateText (Join-Path $target 'config.toml') '' }
  else { $target=Join-Path $target 'new/codex-home' }
  Assert ((Invoke-ModelImport $target $Backend $Domestic) -eq @($Domestic.models).Count)
  $newConfig=Read-PrivateText (Join-Path $target 'config.toml')
  Assert ($newConfig -notmatch '(?m)^model_provider\s*=|^model\s*=|^base_url\s*=|^review_model\s*=')
  if ($mode -ceq 'new_home') { Assert (-not (Test-PrivatePath (Join-Path $target 'auth.json'))) }
}

# A failed final catalog validation restores absence and zero-byte configs separately.
foreach ($missing in @($false,$true)) {
  $target=Setup 'initial-rollback'
  $configPath=Join-Path $target 'config.toml'
  if ($missing) { [IO.File]::Delete($configPath) } else { Write-PrivateText $configPath '' }
  $script:target=$target
  function Invoke-CodexExport([string]$Exe,[string[]]$Arguments,[string]$TargetHome) {
    if ($TargetHome -cne $script:target) {
      $info=Get-ConfigInfo (Read-PrivateText (Join-Path $TargetHome 'config.toml'))
      if ((Get-TomlString $info.Values['model_catalog_json']) -ceq (Join-Path $script:target 'myautoapi-models.json')) { throw 'injected final validation' }
    }
    return (& $script:realExport $Exe $Arguments $TargetHome)
  }
  $failed=$false; try { $null=Invoke-ModelImport $target $Backend $Domestic } catch { $failed=$true }
  Assert $failed
  Assert ((Test-PrivatePath $configPath) -ne $missing)
  if (-not $missing) { Assert (([IO.FileInfo]$configPath).Length -eq 0) }
  Assert (-not (Test-PrivatePath (Join-Path $target 'myautoapi-models.json')))
  Assert (-not (Test-PrivatePath (Join-Path $target 'myautoapi-models.managed.json')))
  Set-Item Function:Invoke-CodexExport $script:realExport
}
# Existing default/review aliases remain valid even when absent from the catalog.
$target=Setup 'custom-default-alias'
$configPath=Join-Path $target 'config.toml'
$aliasConfig="model=`"customer-gpt-alias`"`nreview_model=`"customer-review-alias`"`n"
Write-PrivateText $configPath $aliasConfig
Assert ((Invoke-ModelImport $target $Backend $Domestic) -eq @($Domestic.models).Count)
$aliasInfo=Get-ConfigInfo (Read-PrivateText $configPath)
Assert ((Get-TomlString $aliasInfo.Values['model']) -ceq 'customer-gpt-alias')
Assert ((Get-TomlString $aliasInfo.Values['review_model']) -ceq 'customer-review-alias')
$aliasCatalog=ConvertFrom-Json -InputObject (Read-PrivateText (Join-Path $target 'myautoapi-models.json'))
Assert (@($aliasCatalog.models | Where-Object {$_.slug -ceq 'customer-gpt-alias'}).Count -eq 0)

# A BOM followed immediately by a table header must not move into the TOML body.
$bomText=([string][char]0xFEFF)+"[desktop]`ntheme=`"dark`"`n"
$bomPatched=Set-ModelConfig $bomText (Get-ConfigInfo $bomText) '/synthetic/models.json' @('high','max')
Assert (-not $bomPatched.Contains([string][char]0xFEFF))
Assert ($bomPatched.StartsWith('model_catalog_json'))
$target=Setup 'bom-header'
$configPath=Join-Path $target 'config.toml'
[IO.File]::WriteAllText($configPath,"[desktop]`ntheme=`"dark`"`n",(New-Object Text.UTF8Encoding($true)))
Assert ((Invoke-ModelImport $target $Backend $Domestic) -eq @($Domestic.models).Count)
Assert (-not (Read-PrivateText $configPath).Contains([string][char]0xFEFF))
$null=Invoke-CodexExport $Backend @('debug','models') $target

# Changes during both the initial export and staging validation must survive untouched.
foreach ($timing in @('early','staged')) {
  foreach ($changed in @('config','catalog','managed','source')) {
    $target=Setup ('input-race-'+$timing+'-'+$changed)
    $null=Invoke-ModelImport $target $Backend $Domestic
    $configPath=Join-Path $target 'config.toml'
    $paths=@((Join-Path $target 'config.toml'),(Join-Path $target 'myautoapi-models.json'),(Join-Path $target 'myautoapi-models.managed.json'))
    if ($changed -ceq 'source') {
      $sourcePath=Join-Path $target 'user-original-models.json'
      Write-PrivateText $sourcePath (Read-PrivateText $paths[1])
      $text=Read-PrivateText $configPath; $info=Get-ConfigInfo $text
      Write-PrivateText $configPath (Set-ModelConfig $text $info $sourcePath @('high','max'))
      $paths+=@($sourcePath)
    }
    $index=@{config=0;catalog=1;managed=2;source=3}[$changed]
    $before=@($paths | ForEach-Object { Read-PrivateText $_ })
    $script:target=$target; $script:racePath=$paths[$index]; $script:raceTiming=$timing; $script:raceInjected=$false
    function Invoke-CodexExport([string]$Exe,[string[]]$Arguments,[string]$TargetHome) {
      $shouldChange=if ($script:raceTiming -ceq 'early') { $TargetHome -ceq $script:target -and $Arguments -contains '--bundled' } else { $TargetHome -cne $script:target }
      if ($shouldChange -and -not $script:raceInjected) {
        $script:raceInjected=$true
        [IO.File]::AppendAllText($script:racePath,"`n")
      }
      return (& $script:realExport $Exe $Arguments $TargetHome)
    }
    $failed=$false; $diagnostic=''
    try { $null=Invoke-ModelImport $target $Backend $Domestic } catch { $failed=$true; $diagnostic=Format-ModelImportError $_ }
    Set-Item Function:Invoke-CodexExport $script:realExport
    Assert $script:raceInjected; Assert $failed
    Assert ($diagnostic.Contains('导入过程中被修改'))
    Assert ($diagnostic.Contains($paths[$index]))
    for ($i=0;$i -lt $paths.Count;$i++) {
      $expected=$before[$i]; if ($i -eq $index) { $expected+="`n" }
      Assert ((Read-PrivateText $paths[$i]) -ceq $expected)
    }
    Assert (@(Get-ChildItem -LiteralPath $target -Directory -Force | Where-Object {$_.Name.StartsWith('.myautoapi-models-')}).Count -eq 0)
  }
}

# Missing input paths remain distinct from zero-byte files created during export.
$target=Setup 'input-created-during-export'
$script:target=$target; $script:racePath=Join-Path $target 'myautoapi-models.json'; $script:raceInjected=$false
function Invoke-CodexExport([string]$Exe,[string[]]$Arguments,[string]$TargetHome) {
  if (-not $script:raceInjected -and $TargetHome -ceq $script:target -and $Arguments -contains '--bundled') {
    $script:raceInjected=$true; Write-PrivateText $script:racePath ''
  }
  return (& $script:realExport $Exe $Arguments $TargetHome)
}
$before=Read-PrivateText (Join-Path $target 'config.toml')
$failed=$false; try { $null=Invoke-ModelImport $target $Backend $Domestic } catch { $failed=$true }
Set-Item Function:Invoke-CodexExport $script:realExport
Assert $failed; Assert (Test-PrivatePath $script:racePath)
Assert ((Read-PrivateText $script:racePath) -ceq '')
Assert ((Read-PrivateText (Join-Path $target 'config.toml')) -ceq $before)
Assert (-not (Test-PrivatePath (Join-Path $target 'myautoapi-models.managed.json')))

# Only controlled diagnostics may display messages; raw external text is never echoed.
foreach ($entry in @(
  @{Text='missing field `display_name`; private-credential-do-not-print'; Expected='display_name'},
  @{Text='TOML parse error at line 11, column 3; private-credential-do-not-print'; Expected='第 11 行，第 3 列'},
  @{Text='/synthetic/config.toml:11:3: duplicate key private-credential-do-not-print'; Expected='第 11 行，第 3 列'},
  @{Text='missing field `supports_parallel_tool_calls`; private-credential-do-not-print'; Expected='supports_parallel_tool_calls'},
  @{Text='unrecognized subcommand models; private-credential-do-not-print'; Expected='未识别模型目录导出命令'},
  @{Text='unknown variant private-credential-do-not-print'; Expected='当前 Codex 无法识别的字段值'},
  @{Text='中文外部异常 private-credential-do-not-print'; Expected='Codex 后端命令执行失败'}
)) {
  $diagnostic=''
  try { Invoke-ModelStep '校验测试后端' '/synthetic/backend debug models' { throw (New-ModelBackendError $entry.Text 17) } } catch { $diagnostic=Format-ModelImportError $_ }
  Assert ($diagnostic.Contains($entry.Expected)); Assert ($diagnostic.Contains('后端退出码：17'))
  Assert ($diagnostic.Contains('校验测试后端')); Assert ($diagnostic.Contains('/synthetic/backend'))
  Assert (-not $diagnostic.Contains('private-credential-do-not-print'))
}
foreach ($action in @(
  { throw '中文外部异常 private-credential-do-not-print' },
  { ConvertFrom-ModelJson '{private-credential-do-not-print' '解析合成目录' '/synthetic/catalog.json' }
)) {
  $diagnostic=''
  try { Invoke-ModelStep '读取合成文件' '/synthetic/input.json' $action } catch { $diagnostic=Format-ModelImportError $_ }
  Assert (-not $diagnostic.Contains('private-credential-do-not-print'))
  Assert ($diagnostic.Contains('文件格式或系统操作失败') -or $diagnostic.Contains('不是有效的 JSON'))
  Assert ($diagnostic.Contains('/synthetic/'))
}
foreach ($entry in @(
  @{Type='System.UnauthorizedAccessException'; Expected='权限不足'},
  @{Type='System.IO.FileNotFoundException'; Expected='文件不存在'},
  @{Type='System.IO.DirectoryNotFoundException'; Expected='文件目录不存在'},
  @{Type='System.IO.IOException'; Expected='文件系统读写失败'}
)) {
  $diagnostic=''
  try { Invoke-ModelStep '读取合成文件' '/synthetic/input.json' { throw (New-Object $entry.Type -ArgumentList 'private-credential-do-not-print') } } catch { $diagnostic=Format-ModelImportError $_ }
  Assert ($diagnostic.Contains($entry.Expected)); Assert ($diagnostic.Contains($entry.Type.Split('.')[-1]))
  Assert ($diagnostic.Contains('/synthetic/input.json')); Assert (-not $diagnostic.Contains('private-credential-do-not-print'))
}

# A syntax failure from the optional original-config export must retain its context.
$target=Setup 'existing-toml-syntax'
$configPath=Join-Path $target 'config.toml'
$badToml="web_search=`"disabled`"`nweb_search=`"disabled`"`n"
Write-PrivateText $configPath $badToml
$failed=$false; $diagnostic=''
try { $null=Invoke-ModelImport $target $Backend $Domestic } catch { $failed=$true; $diagnostic=Format-ModelImportError $_ }
Assert $failed; Assert ($diagnostic.Contains('解析已有 TOML 配置'))
Assert ($diagnostic.Contains($configPath)); Assert ($diagnostic.Contains('第 2 行'))
Assert ((Read-PrivateText $configPath) -ceq $badToml)
Assert (-not (Test-PrivatePath (Join-Path $target 'myautoapi-models.json')))

# Capture real stderr/exit status through Process, using an isolated Linux fixture.
if ($env:OS -ne 'Windows_NT') {
  $target=Setup 'backend-diagnostic'
  $badBackend=Join-Path $target 'backend-error.sh'
  Write-PrivateText $badBackend "#!/bin/sh`nprintf '%s\n' 'missing field ``display_name``; private-credential-do-not-print' >&2`nexit 17`n"
  & /bin/chmod 700 $badBackend
  $diagnostic=''
  try { $null=Invoke-CodexExport $badBackend @('debug','models','--bundled') $target } catch { $diagnostic=Format-ModelImportError $_ }
  Assert ($diagnostic.Contains('display_name')); Assert ($diagnostic.Contains('后端退出码：17'))
  Assert ($diagnostic.Contains($badBackend)); Assert (-not $diagnostic.Contains('private-credential-do-not-print'))
}
Write-Output "PowerShell import/rollback: $script:assertions assertions passed"
