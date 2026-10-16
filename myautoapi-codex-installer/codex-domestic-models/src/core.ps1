# Windows PowerShell 5.1 compatible. Keep behavior in sync with core.js via parity tests.
function New-ModelImportError([string]$Message) {
  $problem=New-Object System.Exception -ArgumentList $Message
  $problem.Data['ModelImportSafe']=$true
  return $problem
}

function Invoke-ModelStep([string]$Name, [string]$Target, [scriptblock]$Action) {
  try { & $Action }
  catch {
    if (-not $_.Exception.Data.Contains('ModelImportStep')) { $_.Exception.Data['ModelImportStep']=$Name }
    if ($Target -and -not $_.Exception.Data.Contains('ModelImportTarget')) { $_.Exception.Data['ModelImportTarget']=$Target }
    throw
  }
}

function Format-ModelImportError($Record, [switch]$Uninstall) {
  $problem=if ($Record -is [System.Management.Automation.ErrorRecord]) { $Record.Exception } elseif ($Record -is [Exception]) { $Record } else { $null }
  $reason=$null; $step=$null; $target=$null; $exitStatus=$null; $kind='Error'; $systemCode=$null; $category=$null
  while ($null -ne $problem) {
    if (-not $reason -and $problem.Data['ModelImportSafe'] -eq $true) { $reason=$problem.Message }
    if (-not $step -and $problem.Data['ModelImportStep']) { $step=[string]$problem.Data['ModelImportStep'] }
    if (-not $target -and $problem.Data['ModelImportTarget']) { $target=[string]$problem.Data['ModelImportTarget'] }
    if ($null -eq $exitStatus -and $problem.Data.Contains('ModelImportExitStatus')) { $exitStatus=$problem.Data['ModelImportExitStatus'] }
    if ($problem.GetType().Name -cmatch '^[A-Za-z][A-Za-z0-9]*Exception$') { $kind=$problem.GetType().Name }
    if ($problem -is [System.ComponentModel.Win32Exception]) { $systemCode=$problem.NativeErrorCode }
    if ($problem -is [UnauthorizedAccessException]) { $category='权限不足，无法访问目标文件。' }
    elseif ($problem -is [IO.FileNotFoundException]) { $category='文件不存在。' }
    elseif ($problem -is [IO.DirectoryNotFoundException]) { $category='文件目录不存在。' }
    elseif ($problem -is [IO.PathTooLongException]) { $category='文件路径过长。' }
    elseif ($problem -is [Text.DecoderFallbackException]) { $category='文件包含无效的 UTF-8 字节。' }
    elseif ($problem -is [IO.IOException]) { $category='文件系统读写失败。' }
    $problem=$problem.InnerException
  }
  if (-not $reason) { if (-not $category) { $category='文件格式或系统操作失败。' }; $reason=$category+'（'+$kind+'）' }
  if (-not $step) { $step=if ($Uninstall) { '初始化模型卸载' } else { '初始化模型导入' } }
  $lines=New-Object 'System.Collections.Generic.List[string]'
  $lines.Add($(if ($Uninstall) { '卸载失败' } else { '导入失败' })); $lines.Add('失败步骤：'+$step)
  if ($target) { $lines.Add('文件或命令：'+$target) }
  $lines.Add('原因：'+$reason)
  if ($null -ne $exitStatus) { $lines.Add('后端退出码：'+$exitStatus) }
  if ($null -ne $systemCode) { $lines.Add('系统错误码：'+$systemCode) }
  return ($lines -join "`n")
}

function ConvertFrom-ModelJson([string]$Text, [string]$Step, [string]$Target) {
  return (Invoke-ModelStep $Step $Target {
    try { ConvertFrom-Json -InputObject $Text -ErrorAction Stop }
    catch { throw (New-ModelImportError '内容不是有效的 JSON。') }
  })
}

function New-OrdinalMap {
  return ,(New-Object 'System.Collections.Generic.Dictionary[string,object]' ([System.StringComparer]::Ordinal))
}

function ConvertTo-CanonicalValue($Value) {
  if ($null -eq $Value) { return $null }
  if ($Value -is [Array]) {
    $items=New-Object 'System.Collections.Generic.List[object]'
    foreach ($item in $Value) { $items.Add((ConvertTo-CanonicalValue $item)) }
    return ,$items.ToArray()
  }
  if ($Value -is [pscustomobject]) {
    $result=[ordered]@{}
    foreach ($name in ($Value.PSObject.Properties.Name | Sort-Object -CaseSensitive)) { $result[$name]=ConvertTo-CanonicalValue $Value.$name }
    return $result
  }
  return $Value
}

function Get-TomlString([string]$Raw) {
  $value = $Raw.Trim()
  if ($value -cmatch "^'[^'\r\n]*'$") { return $value.Substring(1, $value.Length - 2) }
  if ($value -cmatch '^"(?:[^"\\\r\n]|\\.)*"$') {
    try {
      $value = [regex]::Replace($value, '\\(?:U[0-9a-fA-F]{8}|e|[\s\S])', {
        param($match)
        $token=$match.Value
        if ($token -ceq '\e') { return '\u001b' }
        if ($token[1] -cne 'U' -or $token.Length -ne 10) { return $token }
        $decoded=[char]::ConvertFromUtf32([Convert]::ToInt32($token.Substring(2),16))
        $encoded=ConvertTo-Json -InputObject $decoded -Compress
        return $encoded.Substring(1,$encoded.Length-2)
      })
      return ($value | ConvertFrom-Json -ErrorAction Stop)
    } catch { throw (New-ModelImportError 'TOML 字符串格式无效。') }
  }
  throw (New-ModelImportError '模型目录、默认模型等目标字段须使用单行 TOML 字符串。')
}

function Get-TomlKeyParts([string]$Raw) {
  $parts = New-Object 'System.Collections.Generic.List[string]'
  $pos = 0
  $pattern = '\s*("(?:[^"\\]|\\.)*"|''[^'']*''|[A-Za-z0-9_-]+)\s*'
  while ($pos -lt $Raw.Length) {
    $match = [regex]::Match($Raw.Substring($pos), '^' + $pattern)
    if (-not $match.Success) { throw (New-ModelImportError '无法识别 TOML 字段或配置段。') }
    $key = $match.Groups[1].Value
    if ($key.StartsWith('"') -or $key.StartsWith("'")) { $key = Get-TomlString $key }
    $parts.Add($key)
    $pos += $match.Length
    if ($pos -eq $Raw.Length) { break }
    if ($Raw[$pos] -cne '.') { throw (New-ModelImportError 'TOML 字段路径无效。') }
    $pos += 1
  }
  if (-not $parts.Count) { throw (New-ModelImportError '空 TOML 字段。') }
  return ,$parts.ToArray()
}

function Get-TomlStatements([string]$Text) {
  $records = New-Object 'System.Collections.Generic.List[object]'
  $depth = New-Object 'System.Collections.Generic.Stack[char]'
  $clean = New-Object System.Text.StringBuilder
  $start = 0; $quote = ''; $triple = $false; $escaped = $false; $comment = $false
  for ($i = 0; $i -lt $Text.Length; $i++) {
    $ch = [string]$Text[$i]
    if ($comment) {
      if ($ch -cne "`n") { [void]$clean.Append(' '); continue }
      $comment = $false
    }
    if ($quote) {
      [void]$clean.Append($ch)
      if ($escaped) { $escaped = $false; continue }
      if ($quote -ceq '"' -and $ch -ceq '\') { $escaped = $true; continue }
      if ($ch -ceq $quote) {
        if (-not $triple) { $quote = '' }
        elseif ($i + 2 -lt $Text.Length -and $Text.Substring($i, 3) -ceq ($quote * 3)) {
          $run = 3
          while ($run -lt 5 -and $i + $run -lt $Text.Length -and [string]$Text[$i + $run] -ceq $quote) { $run += 1 }
          [void]$clean.Append($Text.Substring($i + 1, $run - 1)); $i += $run - 1; $quote = ''; $triple = $false
        }
      } elseif ($ch -ceq "`n" -and -not $triple) { throw (New-ModelImportError 'TOML 字符串未闭合。') }
      continue
    }
    if ($ch -ceq '#') { $comment = $true; [void]$clean.Append(' '); continue }
    if ($ch -ceq '"' -or $ch -ceq "'") {
      $quote = $ch; $triple = $i + 2 -lt $Text.Length -and $Text.Substring($i, 3) -ceq ($ch * 3)
      [void]$clean.Append($ch)
      if ($triple) { [void]$clean.Append($ch * 2); $i += 2 }
      continue
    }
    if ($ch -ceq '[' -or $ch -ceq '{') { $depth.Push([char]$ch) }
    if ($ch -ceq ']' -or $ch -ceq '}') {
      $expected = if ($ch -ceq ']') { '[' } else { '{' }
      if (-not $depth.Count -or [string]$depth.Pop() -cne $expected) { throw (New-ModelImportError 'TOML 括号不匹配。') }
    }
    [void]$clean.Append($ch)
    if ($ch -ceq "`n" -and -not $depth.Count) {
      $records.Add(@{Start=$start; End=$i+1; Code=$clean.ToString()}); $start=$i+1; [void]$clean.Clear()
    }
  }
  if ($quote -or $depth.Count) { throw (New-ModelImportError 'TOML 字符串或括号未闭合。') }
  if ($start -lt $Text.Length) { $records.Add(@{Start=$start; End=$Text.Length; Code=$clean.ToString()}) }
  return ,$records.ToArray()
}

function Get-ConfigInfo([string]$Text, [switch]$ReadOnly) {
  $info = @{Values=(New-OrdinalMap); Edits=(New-OrdinalMap); FirstHeader=$Text.Length; Desktop=$null; DesktopInline=$false; DesktopDotted=$false; ProfileCatalog=$false}
  $scope = @()
  foreach ($rec in (Get-TomlStatements $Text)) {
    $code = $rec.Code.Trim([char[]]@(' ', "`r", "`n", "`t", [char]0xFEFF))
    if (-not $code) { continue }
    if ($code.StartsWith('[')) {
      $header = [regex]::Match($code, '^(\[\[?)([\s\S]*?)(\]\]?)$')
      if (-not $header.Success -or $header.Groups[1].Length -ne $header.Groups[3].Length) { throw (New-ModelImportError 'TOML 配置段无效。') }
      $scope = Get-TomlKeyParts $header.Groups[2].Value.Trim()
      $info.FirstHeader = [Math]::Min($info.FirstHeader, $rec.Start)
      if ($scope.Count -eq 1 -and $scope[0] -ceq 'desktop') {
        if ($header.Groups[1].Value -ceq '[[' -or $null -ne $info.Desktop) { throw (New-ModelImportError 'desktop 配置段重复或格式无效。') }
        $info.Desktop = $rec
      }
      continue
    }
    $q=''; $esc=$false
    for ($eq=0; $eq -lt $rec.Code.Length; $eq++) {
      $ch = [string]$rec.Code[$eq]
      if ($q) {
        if ($esc) { $esc=$false } elseif ($ch -ceq '\' -and $q -ceq '"') { $esc=$true } elseif ($ch -ceq $q) { $q='' }
      } elseif ($ch -ceq '"' -or $ch -ceq "'") { $q=$ch } elseif ($ch -ceq '=') { break }
    }
    if ($eq -eq $rec.Code.Length) { throw (New-ModelImportError 'TOML 配置行无效。') }
    $parts = Get-TomlKeyParts $rec.Code.Substring(0,$eq).Trim().TrimStart([char]0xFEFF)
    $path = @($scope) + @($parts); $key = $path -join '.'
    if (-not $scope.Count -and $parts[0] -ceq 'desktop') {
      $info.DesktopInline = $parts.Count -eq 1
      $info.DesktopDotted = $parts.Count -gt 1 -or $info.DesktopDotted
    }
    if ($path.Count -gt 1 -and $path[0] -ceq 'profiles' -and $path[-1] -ceq 'model_catalog_json') {
      if ($ReadOnly) { $info.ProfileCatalog=$true }
      else { throw (New-ModelImportError 'profile 中另有 model_catalog_json；请先统一目录配置再导入。') }
    }
    if (($path.Count -eq 1 -and @('model_catalog_json','model','review_model','profile','model_provider') -ccontains $key) -or ($path.Count -eq 2 -and $key -ceq 'desktop.enabled-reasoning-efforts')) {
      if ($info.Edits.ContainsKey($key)) { throw (New-ModelImportError '目标 TOML 字段重复。') }
      $vstart = $eq + 1
      while ($vstart -lt $rec.Code.Length -and [char]::IsWhiteSpace($rec.Code[$vstart])) { $vstart += 1 }
      $vend = $rec.Code.TrimEnd().Length
      $info.Edits[$key] = @{Start=$rec.Start+$vstart; End=$rec.Start+$vend}
      $info.Values[$key] = $rec.Code.Substring($vstart,$vend-$vstart)
    }
  }
  if ($info.DesktopInline -and -not $ReadOnly) { throw (New-ModelImportError 'desktop 使用内联表；请改为 [desktop] 配置段后导入。') }
  return $info
}

function Get-TomlStringArray([string]$Raw) {
  $body = $Raw.Trim(); $items = New-Object 'System.Collections.Generic.List[string]'
  if (-not $body.StartsWith('[') -or -not $body.EndsWith(']')) { throw (New-ModelImportError '桌面推理档位必须是字符串数组。') }
  $body = $body.Substring(1,$body.Length-2)
  $pos=0
  while ($pos -lt $body.Length -and $body.Substring($pos).Trim()) {
    $match = [regex]::Match($body.Substring($pos),'^\s*("(?:[^"\\]|\\.)*"|''[^'']*'')\s*(,|$)')
    if (-not $match.Success) { throw (New-ModelImportError '桌面推理档位必须是字符串数组。') }
    $items.Add((Get-TomlString $match.Groups[1].Value)); $pos += $match.Length
  }
  return ,$items.ToArray()
}

function Set-ModelConfig([string]$Text, $Info, [string]$CatalogPath, [string[]]$Efforts) {
  if ($Text.StartsWith([string][char]0xFEFF)) { $Text=$Text.TrimStart([char]0xFEFF); $Info=Get-ConfigInfo $Text }
  $newline = if ($Text.Contains("`r`n")) { "`r`n" } else { "`n" }
  $changes = New-Object 'System.Collections.Generic.List[object]'
  $rootLines = New-Object 'System.Collections.Generic.List[string]'
  $values = @{
    'model_catalog_json'= (ConvertTo-Json -InputObject $CatalogPath -Compress)
    'desktop.enabled-reasoning-efforts'= (ConvertTo-Json -InputObject @($Efforts) -Compress)
  }
  foreach ($key in @('model_catalog_json','desktop.enabled-reasoning-efforts')) {
    if ($Info.Edits.ContainsKey($key)) { $changes.Add(@{Start=$Info.Edits[$key].Start; End=$Info.Edits[$key].End; Value=$values[$key]}); continue }
    if ($key -ceq 'model_catalog_json') { $rootLines.Add('model_catalog_json = ' + $values[$key]); continue }
    $line = 'enabled-reasoning-efforts = ' + $values[$key] + $newline
    if ($null -ne $Info.Desktop) {
      $prefix = if ($Text[$Info.Desktop.End-1] -ceq "`n") { '' } else { $newline }
      $changes.Add(@{Start=$Info.Desktop.End; End=$Info.Desktop.End; Value=$prefix+$line})
    } elseif ($Info.DesktopDotted) { $rootLines.Add('desktop.'+$line.Trim()) }
    else {
      $prefix = if ($Text -and -not $Text.EndsWith("`n")) { $newline } else { '' }
      $changes.Add(@{Start=$Text.Length; End=$Text.Length; Value=$prefix+$newline+'[desktop]'+$newline+$line})
    }
  }
  if ($rootLines.Count) {
    $at = $Info.FirstHeader
    $prefix = if ($at -and $Text[$at-1] -cne "`n") { $newline } else { '' }
    $changes.Add(@{Start=$at; End=$at; Value=$prefix+($rootLines -join $newline)+$newline})
  }
  for ($index=0; $index -lt $changes.Count; $index++) { $changes[$index].Order=$index }
  foreach ($change in ($changes | Sort-Object @{Expression='Start';Descending=$true},@{Expression='End';Descending=$true},@{Expression='Order';Descending=$false})) {
    $Text = $Text.Substring(0,$change.Start) + $change.Value + $Text.Substring($change.End)
  }
  $null = Get-ConfigInfo $Text
  return $Text
}

function Test-ModelCatalog($Catalog) {
  if ($null -eq $Catalog -or $Catalog.models -isnot [Array] -or -not $Catalog.models.Count) { throw (New-ModelImportError '模型目录必须包含非空 models 数组。') }
  $seen = New-OrdinalMap
  foreach ($model in $Catalog.models) {
    if ($null -eq $model -or $model.slug -isnot [string] -or -not $model.slug -or $seen.ContainsKey($model.slug)) { throw (New-ModelImportError '模型目录中有空 ID 或重复 ID。') }
    $seen[$model.slug]=$true
    if ($model.display_name -isnot [string] -or $model.supported_reasoning_levels -isnot [Array] -or -not $model.shell_type -or -not $model.visibility -or $model.supported_in_api -isnot [bool] -or $null -eq $model.priority -or $model.support_verbosity -isnot [bool] -or $null -eq $model.truncation_policy -or $model.experimental_supported_tools -isnot [Array]) { throw (New-ModelImportError '模型目录缺少 Codex 必需元数据。') }
    if ($model.model_messages.instructions_template -isnot [string] -and $model.base_instructions -isnot [string]) { throw (New-ModelImportError '模型目录缺少模型指令。') }
  }
}

function Merge-ModelCatalog($Existing, $App, $Domestic, $Managed, $Info) {
  Test-ModelCatalog $App; Test-ModelCatalog $Domestic
  if ($null -ne $Existing) { Test-ModelCatalog $Existing }
  if ($Managed -isnot [Array]) { throw (New-ModelImportError '已管理模型记录格式无效。') }
  foreach ($id in $Managed) {
    if ($id -isnot [string] -or -not $id) { throw (New-ModelImportError '已管理模型记录格式无效。') }
    if ($id -match '^gpt-') { throw (New-ModelImportError '管理记录包含 GPT，停止导入以保留 GPT。') }
  }
  $byId=New-OrdinalMap; $owned=New-OrdinalMap; $order=New-Object 'System.Collections.Generic.List[string]'
  foreach ($id in (@($Managed) + @($Domestic.models | ForEach-Object {$_.slug}))) { $owned[$id]=$true }
  $initial = @(); if ($null -ne $Existing) { $initial=@($Existing.models) }
  foreach ($model in (@($initial) + @($App.models))) {
    if (-not $byId.ContainsKey($model.slug) -and -not $owned.ContainsKey($model.slug)) { $byId[$model.slug]=$model; $order.Add($model.slug) }
  }
  foreach ($model in $Domestic.models) { $byId[$model.slug]=$model; $order.Add($model.slug) }
  $source = if ($null -ne $Existing) { $Existing } else { $App }
  $result = ConvertFrom-Json -InputObject (ConvertTo-Json -InputObject $source -Depth 100)
  $result.models = @($order | ForEach-Object {$byId[$_]})
  if (-not @($result.models | Where-Object {$_.slug -match '^gpt-'}).Count) { throw (New-ModelImportError '未取得 GPT 完整目录，停止导入。') }
  $levels = New-OrdinalMap
  $efforts = @()
  if ($Info.Values.ContainsKey('desktop.enabled-reasoning-efforts')) { $efforts = Get-TomlStringArray $Info.Values['desktop.enabled-reasoning-efforts'] }
  else { $efforts=@($result.models | Where-Object {$_.slug -match '^gpt-'} | ForEach-Object {$_.supported_reasoning_levels} | ForEach-Object {$_.effort}) }
  $efforts += @($Domestic.models | ForEach-Object {$_.supported_reasoning_levels} | ForEach-Object {$_.effort})
  $effortOrder = @('none','minimal','low','medium','high','xhigh','ultra','max')
  foreach ($level in $efforts) {
    if ($effortOrder -cnotcontains $level) { throw (New-ModelImportError '桌面档位存在未知值，停止导入。') }
    $levels[$level]=$true
  }
  Test-ModelCatalog $result
  return @{Catalog=$result; Efforts=@($effortOrder | Where-Object {$levels.ContainsKey($_)}); Managed=@($Domestic.models | ForEach-Object {$_.slug})}
}

function Invoke-ModelImport([string]$TargetHome, [string]$Backend, $Domestic) {
  $configPath = Join-Path $TargetHome 'config.toml'
  $catalogPath = Join-Path $TargetHome 'myautoapi-models.json'
  $managedPath = Join-Path $TargetHome 'myautoapi-models.managed.json'
  Invoke-ModelStep '创建 Codex 配置目录' $TargetHome { if (-not (Test-PrivatePath $TargetHome)) { $null=[IO.Directory]::CreateDirectory($TargetHome) } }
  $readSet=New-Object 'System.Collections.Generic.List[object]'
  $original=Read-ModelInput $readSet $configPath '读取 Codex 配置'
  $outputCatalog=Read-ModelInput $readSet $catalogPath '读取原输出模型目录'
  $rawManaged=Read-ModelInput $readSet $managedPath '读取已管理模型记录'
  $info=Invoke-ModelStep '解析 Codex 配置' $configPath { Get-ConfigInfo ([string]$original) }
  $existing=$null; $managed=@(); $source=$null
  if ($info.Values.ContainsKey('model_catalog_json')) {
    $source=Invoke-ModelStep '解析原模型目录路径' $configPath {
      $value=Get-TomlString $info.Values['model_catalog_json']
      if ($value.StartsWith('~/') -or $value.StartsWith('~\')) { $value=Join-Path $HOME $value.Substring(2) }
      if (-not [IO.Path]::IsPathRooted($value)) { $value=Join-Path $TargetHome $value }
      [IO.Path]::GetFullPath($value)
    }
    $sourceText=Read-ModelInput $readSet $source '读取原模型目录'
    if ($null -ne $sourceText) {
      $existing=ConvertFrom-ModelJson $sourceText '解析原模型目录' $source
      Invoke-ModelStep '校验原模型目录' $source { Test-ModelCatalog $existing }
    }
  }
  if ($null -ne $rawManaged) {
    $managed=Invoke-ModelStep '解析已管理模型记录' $managedPath {
      if (-not $rawManaged.TrimStart().StartsWith('[')) { throw (New-ModelImportError '已管理模型记录格式无效。') }
      ,@(ConvertFrom-ModelJson $rawManaged '解析已管理模型记录' $managedPath)
    }
  }
  $app=Invoke-ModelStep '导出并解析 Codex 内置模型目录' ($Backend+' debug models --bundled') {
    ConvertFrom-ModelJson (Invoke-CodexExport $Backend @('debug','models','--bundled') $TargetHome) '解析 Codex 内置模型目录' ($Backend+' debug models --bundled')
  }
  if ($null -eq $existing) {
    try { $candidate=ConvertFrom-Json -InputObject (Invoke-CodexExport $Backend @('debug','models') $TargetHome); Test-ModelCatalog $candidate; $existing=$candidate }
    catch {
      if ($_.Exception.Data['ConfigSyntax'] -eq $true) {
        $_.Exception.Data['ModelImportStep']='解析已有 TOML 配置'; $_.Exception.Data['ModelImportTarget']=$configPath
        throw
      }
    }
  }
  $merged=Invoke-ModelStep '校验并合并模型元数据' $catalogPath { Merge-ModelCatalog $existing $app $Domestic $managed $info }
  $config=Invoke-ModelStep '合并模型目录指向和桌面档位' $configPath { Set-ModelConfig ([string]$original) $info $catalogPath $merged.Efforts }
  $files = @(
    @{Path=$catalogPath; Text=(ConvertTo-Json -InputObject $merged.Catalog -Depth 100)+"`n"},
    @{Path=$managedPath; Text=(ConvertTo-Json -InputObject @($merged.Managed) -Depth 100)+"`n"},
    @{Path=$configPath; Text=$config}
  )
  Invoke-ModelStep '写入并校验导入结果' $catalogPath { Write-ModelTransaction $files $Backend $TargetHome $original $readSet.ToArray() }
  return $merged.Managed.Count
}

function Remove-ImportedModels([string]$TargetHome) {
  $configPath=Join-Path $TargetHome 'config.toml'
  $catalogPath=Join-Path $TargetHome 'myautoapi-models.json'
  $managedPath=Join-Path $TargetHome 'myautoapi-models.managed.json'
  $readSet=New-Object 'System.Collections.Generic.List[object]'
  $rawManaged=Read-ModelInput $readSet $managedPath '读取已管理模型记录'
  if ($null -eq $rawManaged) { return @{Changed=$false; Active=$false; Count=0} }
  $managed=Invoke-ModelStep '解析已管理模型记录' $managedPath {
    if (-not $rawManaged.TrimStart().StartsWith('[')) { throw (New-ModelImportError '已管理模型记录格式无效。') }
    $null=ConvertFrom-ModelJson $rawManaged '解析已管理模型记录' $managedPath
    # A wrapper preserves nested/empty arrays even when PS 5.1 enumerates the root array.
    $wrapper=ConvertFrom-ModelJson ('{"ids":'+$rawManaged+'}') '解析已管理模型记录' $managedPath
    ,$wrapper.ids
  }
  $owned=New-OrdinalMap
  Invoke-ModelStep '校验已管理模型记录' $managedPath {
    foreach ($id in $managed) {
      if ($id -isnot [string] -or -not $id) { throw (New-ModelImportError '已管理模型记录格式无效。') }
      if ($id -match '^gpt-') { throw (New-ModelImportError '管理记录包含 GPT，停止卸载以保留 GPT。') }
      $owned[$id]=$true
    }
  }
  if (-not $owned.Count) { return @{Changed=$false; Active=$false; Count=0} }
  $catalogText=Invoke-ModelStep '读取原导入模型目录' $catalogPath {
    $text=Read-ModelInput $readSet $catalogPath '读取原导入模型目录'
    if ($null -eq $text) { throw (New-ModelImportError '原导入模型目录不存在，停止卸载。') }
    $text
  }
  $catalog=ConvertFrom-ModelJson $catalogText '解析原导入模型目录' $catalogPath
  Invoke-ModelStep '校验原导入模型目录' $catalogPath { Test-ModelCatalog $catalog }
  $beforeCount=$catalog.models.Count
  $catalog.models=@($catalog.models | Where-Object { -not $owned.ContainsKey($_.slug) })
  Invoke-ModelStep '校验卸载后的模型目录' $catalogPath { Test-ModelCatalog $catalog }
  $count=$beforeCount-$catalog.models.Count
  $original=Read-ModelInput $readSet $configPath '读取 Codex 配置'
  $info=Invoke-ModelStep '解析 Codex 配置' $configPath { Get-ConfigInfo ([string]$original) -ReadOnly }
  $active=$false
  if ($count -gt 0 -and -not $info.ProfileCatalog -and $info.Values.ContainsKey('model_catalog_json')) {
    $source=Invoke-ModelStep '解析当前模型目录路径' $configPath {
      $value=Get-TomlString $info.Values['model_catalog_json']
      if ($value.StartsWith('~/') -or $value.StartsWith('~\')) { $value=Join-Path $HOME $value.Substring(2) }
      if (-not [IO.Path]::IsPathRooted($value)) { $value=Join-Path $TargetHome $value }
      [IO.Path]::GetFullPath($value)
    }
    $active=[StringComparer]::OrdinalIgnoreCase.Equals($source,[IO.Path]::GetFullPath($catalogPath))
  }
  $files=@(
    @{Path=$catalogPath; Text=(ConvertTo-Json -InputObject $catalog -Depth 100)+"`n"},
    @{Path=$managedPath; Text="[]`n"},
    @{Path=$configPath; Text=$original; Keep=$true}
  )
  Invoke-ModelStep '写入并校验卸载结果' $catalogPath { Write-ModelTransaction $files '' $TargetHome $original $readSet.ToArray() }
  return @{Changed=$true; Active=$active; Count=$count}
}
