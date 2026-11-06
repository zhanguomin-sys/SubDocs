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

function ConvertTo-TomlString([string]$Value) {
  # Windows paths are literal TOML strings, so backslashes never become escapes.
  if ($Value.Contains('\') -and -not $Value.Contains("'") -and $Value -notmatch '[\x00-\x1f\x7f]') { return "'"+$Value+"'" }
  $encoded=New-Object System.Text.StringBuilder
  [void]$encoded.Append('"')
  foreach ($character in $Value.ToCharArray()) {
    $number=[int]$character
    if ($number -eq 34) { [void]$encoded.Append('\"') }
    elseif ($number -eq 92) { [void]$encoded.Append('\\') }
    elseif ($number -lt 32 -or $number -eq 127) { [void]$encoded.Append(('\u{0:x4}' -f $number)) }
    else { [void]$encoded.Append($character) }
  }
  [void]$encoded.Append('"')
  return $encoded.ToString()
}

function ConvertTo-TomlStringArray([string[]]$Values) {
  if ($null -eq $Values -or -not $Values.Count) { return '[]' }
  return '['+((@($Values) | ForEach-Object { ConvertTo-TomlString $_ }) -join ',')+']'
}

function New-TomlSyntaxError([string]$Message, [string]$Text, [int]$Offset) {
  $prefix=$Text.Substring(0,[Math]::Min($Offset,$Text.Length))
  $line=1; foreach ($character in $prefix.ToCharArray()) { if ([int]$character -eq 10) { $line++ } }
  $column=$prefix.Length-$prefix.LastIndexOf("`n")
  return (New-ModelImportError ($Message+' 第 '+$line+' 行，第 '+$column+' 列。'))
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
  $start = 0; $quote = ''; $quoteStart = 0; $triple = $false; $escaped = $false; $comment = $false
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
      } elseif ($ch -ceq "`n" -and -not $triple) { throw (New-TomlSyntaxError 'TOML 字符串未闭合。' $Text $quoteStart) }
      continue
    }
    if ($ch -ceq '#') { $comment = $true; [void]$clean.Append(' '); continue }
    if ($ch -ceq '"' -or $ch -ceq "'") {
      $quote = $ch; $quoteStart = $i; $triple = $i + 2 -lt $Text.Length -and $Text.Substring($i, 3) -ceq ($ch * 3)
      [void]$clean.Append($ch)
      if ($triple) { [void]$clean.Append($ch * 2); $i += 2 }
      continue
    }
    if ($ch -ceq '[' -or $ch -ceq '{') { $depth.Push([char]$ch) }
    if ($ch -ceq ']' -or $ch -ceq '}') {
      $expected = if ($ch -ceq ']') { '[' } else { '{' }
      if (-not $depth.Count -or [string]$depth.Pop() -cne $expected) { throw (New-TomlSyntaxError 'TOML 括号不匹配。' $Text $i) }
    }
    [void]$clean.Append($ch)
    if ($ch -ceq "`n" -and -not $depth.Count) {
      $records.Add(@{Start=$start; End=$i+1; Code=$clean.ToString()}); $start=$i+1; [void]$clean.Clear()
    }
  }
  if ($quote) { throw (New-TomlSyntaxError 'TOML 字符串未闭合。' $Text $quoteStart) }
  if ($depth.Count) { throw (New-TomlSyntaxError 'TOML 括号未闭合。' $Text $Text.Length) }
  if ($start -lt $Text.Length) { $records.Add(@{Start=$start; End=$Text.Length; Code=$clean.ToString()}) }
  return ,$records.ToArray()
}


# Reads config.toml statement by statement. It records where the two fields this
# tool owns live (by statement index), so edits never depend on shifting offsets.
function Get-ConfigInfo([string]$Text) {
  $info = @{Values=(New-OrdinalMap); Edits=(New-OrdinalMap); Records=@(); FirstHeaderIndex=-1; DesktopIndex=-1; DesktopInline=$false; DesktopDotted=$false}
  $scope = @()
  $info.Records = Get-TomlStatements $Text
  for ($index = 0; $index -lt $info.Records.Count; $index++) {
    $rec = $info.Records[$index]
    $code = $rec.Code.Trim([char[]]@(' ', "`r", "`n", "`t", [char]0xFEFF))
    if (-not $code) { continue }
    if ($code.StartsWith('[')) {
      $header = [regex]::Match($code, '^(\[\[?)([\s\S]*?)(\]\]?)$')
      if (-not $header.Success -or $header.Groups[1].Length -ne $header.Groups[3].Length) { throw (New-ModelImportError 'TOML 配置段无效。') }
      $scope = Get-TomlKeyParts $header.Groups[2].Value.Trim()
      if ($info.FirstHeaderIndex -lt 0) { $info.FirstHeaderIndex = $index }
      if ($scope.Count -eq 1 -and $scope[0] -ceq 'desktop') {
        if ($header.Groups[1].Value -ceq '[[' -or $info.DesktopIndex -ge 0) { throw (New-ModelImportError 'desktop 配置段重复或格式无效。') }
        $info.DesktopIndex = $index
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
      if ($parts.Count -eq 1) { $info.DesktopInline = $true } else { $info.DesktopDotted = $true }
    }
    if ($path.Count -gt 1 -and $path[0] -ceq 'profiles' -and $path[-1] -ceq 'model_catalog_json') {
      throw (New-ModelImportError 'profile 中另有 model_catalog_json；请先统一目录配置再导入。')
    }
    if (($path.Count -eq 1 -and $path[0] -ceq 'model_catalog_json') -or ($path.Count -eq 2 -and $key -ceq 'desktop.enabled-reasoning-efforts')) {
      if ($info.Edits.ContainsKey($key)) { throw (New-ModelImportError '目标 TOML 字段重复。') }
      $vstart = $eq + 1
      while ($vstart -lt $rec.Code.Length -and [char]::IsWhiteSpace($rec.Code[$vstart])) { $vstart += 1 }
      $vend = $rec.Code.TrimEnd().Length
      if ($vend -le $vstart) { throw (New-ModelImportError ('TOML 字段 '+$key+' 缺少值。')) }
      # Offsets are relative to this statement's own text.
      $info.Edits[$key] = @{Index=$index; Start=$vstart; End=$vend}
      $info.Values[$key] = $rec.Code.Substring($vstart,$vend-$vstart)
    }
  }
  if ($info.DesktopInline) { throw (New-ModelImportError 'desktop 使用内联表；请改为 [desktop] 配置段后导入。') }
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

# Rewrites only the two fields this tool owns. Every other statement is copied
# back byte for byte; edits are keyed by statement index and never sorted.
function Set-ModelConfig([string]$Text, $Info, [string]$CatalogPath, [string[]]$Efforts) {
  $newline = if ($Text.Contains("`r`n")) { "`r`n" } else { "`n" }
  $records = $Info.Records
  if ($records.Count -and $records[$records.Count-1].End -ne $Text.Length) { throw (New-ModelImportError 'TOML 配置解析结果与文本不一致。') }
  $pieces = New-Object 'System.Collections.Generic.List[string]'
  foreach ($rec in $records) { $pieces.Add($Text.Substring($rec.Start, $rec.End - $rec.Start)) }
  $before = New-Object 'System.Collections.Generic.Dictionary[int,string]'
  $after = New-Object 'System.Collections.Generic.Dictionary[int,string]'
  $rootLines = New-Object 'System.Collections.Generic.List[string]'
  $tail = ''
  $catalogValue = ConvertTo-TomlString $CatalogPath
  $effortsValue = ConvertTo-TomlStringArray $Efforts

  foreach ($item in @(@('model_catalog_json',$catalogValue), @('desktop.enabled-reasoning-efforts',$effortsValue))) {
    if ($Info.Edits.ContainsKey($item[0])) {
      $edit = $Info.Edits[$item[0]]; $piece = $pieces[$edit.Index]
      $pieces[$edit.Index] = $piece.Substring(0,$edit.Start) + $item[1] + $piece.Substring($edit.End)
    }
  }
  if (-not $Info.Edits.ContainsKey('model_catalog_json')) { $rootLines.Add('model_catalog_json = ' + $catalogValue) }
  $desktopBlock = ''
  if (-not $Info.Edits.ContainsKey('desktop.enabled-reasoning-efforts')) {
    $line = 'enabled-reasoning-efforts = ' + $effortsValue + $newline
    if ($Info.DesktopIndex -ge 0) {
      $lead = if ($pieces[$Info.DesktopIndex].EndsWith("`n", [StringComparison]::Ordinal)) { '' } else { $newline }
      $after[$Info.DesktopIndex] = $lead + $line
    } elseif ($Info.DesktopDotted) { $rootLines.Add('desktop.' + $line.TrimEnd()) }
    else { $desktopBlock = '[desktop]' + $newline + $line }
  }
  $rootText = ''
  if ($rootLines.Count) { $rootText = ($rootLines -join $newline) + $newline }
  if ($rootText) {
    if ($Info.FirstHeaderIndex -ge 0) { $before[$Info.FirstHeaderIndex] = $rootText } else { $tail += $rootText }
  }
  if ($desktopBlock) { $tail += $(if ($Text -or $tail) { $newline } else { '' }) + $desktopBlock }

  $out = New-Object System.Text.StringBuilder
  for ($index = 0; $index -lt $pieces.Count; $index++) {
    if ($before.ContainsKey($index)) { [void]$out.Append($before[$index]) }
    [void]$out.Append($pieces[$index])
    if ($after.ContainsKey($index)) { [void]$out.Append($after[$index]) }
  }
  if ($tail) {
    if ($out.Length -and $out[$out.Length-1] -cne "`n") { [void]$out.Append($newline) }
    [void]$out.Append($tail)
  }
  $result = $out.ToString()

  # The new text must parse and must carry exactly the values we meant to write.
  $check = Get-ConfigInfo $result
  $written = $check.Values['model_catalog_json']
  $writtenEfforts = $check.Values['desktop.enabled-reasoning-efforts']
  $writtenList = if ($null -ne $writtenEfforts) { Get-TomlStringArray $writtenEfforts } else { $null }
  if ($null -eq $written -or $null -eq $writtenList -or (Get-TomlString $written) -cne $CatalogPath -or ($writtenList -join ',') -cne (@($Efforts) -join ',')) {
    throw (New-ModelImportError '写入后的配置字段与预期不一致，已停止。')
  }
  return $result
}

# Entries from the user's catalog, the Codex cache and the bundled snapshot only
# need to look like model objects; Codex itself decides what it can load.
function Test-ModelEntry($Model) {
  return ($null -ne $Model -and $Model.slug -is [string] -and [bool]$Model.slug -and $Model.display_name -is [string] -and [bool]$Model.visibility)
}

function Test-ModelCatalogShape($Catalog) {
  if ($null -eq $Catalog -or $Catalog.models -isnot [Array] -or -not $Catalog.models.Count) { throw (New-ModelImportError '模型目录必须包含非空 models 数组。') }
  $seen = New-OrdinalMap
  foreach ($model in $Catalog.models) {
    if ($null -eq $model -or $model.slug -isnot [string] -or -not $model.slug -or $seen.ContainsKey($model.slug)) { throw (New-ModelImportError '模型目录中有空 ID 或重复 ID。') }
    $seen[$model.slug] = $true
  }
}

# Full field check, used for the data this tool ships (domestic list and snapshot).
function Test-ModelCatalog($Catalog) {
  Test-ModelCatalogShape $Catalog
  foreach ($model in $Catalog.models) {
    if ($model.display_name -isnot [string] -or $model.supported_reasoning_levels -isnot [Array] -or -not $model.shell_type -or -not $model.visibility -or $model.supported_in_api -isnot [bool] -or $null -eq $model.priority -or $model.support_verbosity -isnot [bool] -or $null -eq $model.truncation_policy -or $model.experimental_supported_tools -isnot [Array]) { throw (New-ModelImportError '模型目录缺少 Codex 必需元数据。') }
    if ($model.model_messages.instructions_template -isnot [string] -and $model.base_instructions -isnot [string]) { throw (New-ModelImportError '模型目录缺少模型指令。') }
  }
}

function Test-CacheNotOlder($CacheVersion, [string]$SnapshotVersion) {
  try {
    $cache = [version](([string]$CacheVersion) -replace '[-+].*$','')
    $snapshot = [version]($SnapshotVersion -replace '[-+].*$','')
    return ($cache -ge $snapshot)
  } catch {
    # The cache is written by the installed Codex. A missing or unreadable version
    # means its format changed, which is what a newer Codex looks like.
    return $true
  }
}

# Official models (including hidden ones such as codex-auto-review) come from the
# bundled snapshot, refreshed by Codex's own models_cache.json when it is usable.
# A cache older than the snapshot only fills in missing models.
function Get-OfficialModels($Snapshot, [string]$SnapshotVersion, $CacheText) {
  $entries = New-OrdinalMap; $order = New-Object 'System.Collections.Generic.List[string]'
  foreach ($model in $Snapshot.models) { $entries[$model.slug] = @{Model=$model; Source='snapshot'}; $order.Add($model.slug) }
  $usedCache = $false
  if ($null -ne $CacheText) {
    try {
      $cache = ConvertFrom-Json -InputObject $CacheText -ErrorAction Stop
      if ($cache.models -is [Array]) {
        $newer = Test-CacheNotOlder $cache.client_version $SnapshotVersion
        foreach ($model in $cache.models) {
          if (-not (Test-ModelEntry $model)) { continue }
          if (-not $entries.ContainsKey($model.slug)) { $entries[$model.slug] = @{Model=$model; Source='cache'}; $order.Add($model.slug); $usedCache = $true }
          elseif ($newer) { $entries[$model.slug] = @{Model=$model; Source='cache'}; $usedCache = $true }
        }
      }
    } catch { $usedCache = $false; $entries = $null }
    if ($null -eq $entries) { return (Get-OfficialModels $Snapshot $SnapshotVersion $null) }
  }
  return @{Entries=$entries; Order=$order; UsedCache=$usedCache}
}

function Merge-ModelCatalog($Existing, $Official, $Domestic, $Managed, $Info) {
  Test-ModelCatalog $Domestic
  if ($null -ne $Existing) { Test-ModelCatalogShape $Existing }
  $owned = New-OrdinalMap
  foreach ($id in @($Managed)) { if ($id -is [string] -and $id -and -not $Official.Entries.ContainsKey($id) -and $id -notmatch '^gpt-') { $owned[$id] = $true } }
  foreach ($model in $Domestic.models) { $owned[$model.slug] = $true }
  $byId = New-OrdinalMap; $order = New-Object 'System.Collections.Generic.List[string]'
  if ($null -ne $Existing) {
    foreach ($model in $Existing.models) {
      if ($owned.ContainsKey($model.slug)) { continue }
      $item = $model
      if ($Official.Entries.ContainsKey($model.slug) -and $Official.Entries[$model.slug].Source -ceq 'cache') { $item = $Official.Entries[$model.slug].Model }
      $byId[$model.slug] = $item; $order.Add($model.slug)
    }
  }
  foreach ($slug in $Official.Order) {
    if (-not $byId.ContainsKey($slug) -and -not $owned.ContainsKey($slug)) { $byId[$slug] = $Official.Entries[$slug].Model; $order.Add($slug) }
  }
  foreach ($model in $Domestic.models) { $byId[$model.slug] = $model; $order.Add($model.slug) }
  if ($null -ne $Existing) { $result = ConvertFrom-Json -InputObject (ConvertTo-Json -InputObject $Existing -Depth 100) }
  else { $result = [pscustomobject]@{models=@()} }
  $result | Add-Member -NotePropertyName models -NotePropertyValue @($order | ForEach-Object { $byId[$_] }) -Force
  if (-not @($result.models | Where-Object { $_.slug -match '^gpt-' }).Count) { throw (New-ModelImportError '未取得 GPT 完整目录，停止导入。') }
  Test-ModelCatalogShape $result

  $efforts = @()
  if ($Info.Values.ContainsKey('desktop.enabled-reasoning-efforts')) { $efforts = Get-TomlStringArray $Info.Values['desktop.enabled-reasoning-efforts'] }
  else { $efforts = @($result.models | Where-Object { $_.slug -match '^gpt-' } | ForEach-Object { $_.supported_reasoning_levels } | ForEach-Object { $_.effort }) }
  $efforts += @($Domestic.models | ForEach-Object { $_.supported_reasoning_levels } | ForEach-Object { $_.effort })
  $effortOrder = @('none','minimal','low','medium','high','xhigh','ultra','max')
  $levels = New-OrdinalMap; $extra = New-Object 'System.Collections.Generic.List[string]'
  foreach ($level in $efforts) {
    if ($level -isnot [string] -or -not $level) { continue }
    if ($levels.ContainsKey($level)) { continue }
    $levels[$level] = $true
    # A level this tool does not know is kept as the user or Codex wrote it.
    if ($effortOrder -cnotcontains $level) { $extra.Add($level) }
  }
  $finalEfforts = @($effortOrder | Where-Object { $levels.ContainsKey($_) }) + @($extra.ToArray())
  return @{Catalog=$result; Efforts=$finalEfforts; Managed=@($Domestic.models | ForEach-Object { $_.slug })}
}

function Get-ConfiguredCatalogPath($Info, [string]$TargetHome) {
  $value = Get-TomlString $Info.Values['model_catalog_json']
  if ($value.StartsWith('~/') -or $value.StartsWith('~\')) { $value = Join-Path $HOME $value.Substring(2) }
  if (-not [IO.Path]::IsPathRooted($value)) { $value = Join-Path $TargetHome $value }
  return [IO.Path]::GetFullPath($value)
}

function ConvertFrom-ManagedIds([string]$Raw) {
  # Ownership hint only. A damaged record falls back to the built-in domestic list.
  if ($null -eq $Raw) { return @() }
  try {
    $wrapper = ConvertFrom-Json -InputObject ('{"ids":' + $Raw + '}') -ErrorAction Stop
    return @($wrapper.ids | Where-Object { $_ -is [string] -and $_ })
  } catch { return @() }
}

function Invoke-ModelImport([string]$TargetHome, $Domestic) {
  $configPath = Join-Path $TargetHome 'config.toml'
  $catalogPath = Join-Path $TargetHome 'myautoapi-models.json'
  $managedPath = Join-Path $TargetHome 'myautoapi-models.managed.json'
  $cachePath = Join-Path $TargetHome 'models_cache.json'
  Invoke-ModelStep '创建 Codex 配置目录' $TargetHome { $null=[IO.Directory]::CreateDirectory($TargetHome) }
  $original = Read-OptionalText $configPath '读取 Codex 配置'
  $info = Invoke-ModelStep '解析 Codex 配置' $configPath { Get-ConfigInfo ([string]$original) }
  $existing = $null
  if ($info.Values.ContainsKey('model_catalog_json')) {
    $source = Invoke-ModelStep '解析原模型目录路径' $configPath { Get-ConfiguredCatalogPath $info $TargetHome }
    $sourceText = Read-OptionalText $source '读取原模型目录'
    if ($null -ne $sourceText) { $existing = ConvertFrom-ModelJson $sourceText '解析原模型目录' $source }
  }
  if ($null -eq $existing) {
    $outputText = Read-OptionalText $catalogPath '读取原输出模型目录'
    if ($null -ne $outputText) { $existing = ConvertFrom-ModelJson $outputText '解析原输出模型目录' $catalogPath }
  }
  $managed = ConvertFrom-ManagedIds (Read-OptionalText $managedPath '读取已管理模型记录')
  # The Codex cache is optional: any read or format problem simply means the snapshot is used.
  $cacheText = $null
  try { $cacheText = Read-OptionalText $cachePath '读取本地模型缓存' } catch { $cacheText = $null }
  $official = Get-OfficialModels $BundledOfficialCatalog $BundledOfficialVersion $cacheText
  $merged = Invoke-ModelStep '校验并合并模型元数据' $catalogPath { Merge-ModelCatalog $existing $official $Domestic $managed $info }
  $config = Invoke-ModelStep '合并模型目录指向和桌面档位' $configPath { Set-ModelConfig ([string]$original) $info $catalogPath $merged.Efforts }
  $files = @(
    @{Path=$catalogPath; Text=(ConvertTo-Json -InputObject $merged.Catalog -Depth 100)+"`n"; Kind='catalog'},
    @{Path=$managedPath; Text=(ConvertTo-Json -InputObject @($merged.Managed) -Depth 100)+"`n"; Kind='managed'},
    @{Path=$configPath; Text=$config; Kind='config'}
  )
  Invoke-ModelStep '写入并校验导入结果' $catalogPath { Write-ModelFiles $files $original }
  return @{Count=$merged.Managed.Count; UsedCache=$official.UsedCache}
}

function Remove-ImportedModels([string]$TargetHome, $Domestic) {
  $catalogPath = Join-Path $TargetHome 'myautoapi-models.json'
  $managedPath = Join-Path $TargetHome 'myautoapi-models.managed.json'
  $catalogText = Read-OptionalText $catalogPath '读取原导入模型目录'
  if ($null -eq $catalogText) { return @{Changed=$false; Active=$false; Count=0} }
  $catalog = ConvertFrom-ModelJson $catalogText '解析原导入模型目录' $catalogPath
  Invoke-ModelStep '校验原导入模型目录' $catalogPath { Test-ModelCatalogShape $catalog }
  $owned = New-OrdinalMap
  foreach ($id in (ConvertFrom-ManagedIds (Read-OptionalText $managedPath '读取已管理模型记录'))) { if ($id -notmatch '^gpt-') { $owned[$id] = $true } }
  foreach ($model in $Domestic.models) { $owned[$model.slug] = $true }
  $remaining = @($catalog.models | Where-Object { -not $owned.ContainsKey($_.slug) })
  $count = $catalog.models.Count - $remaining.Count
  if ($count -eq 0) { return @{Changed=$false; Active=$false; Count=0} }
  $catalog | Add-Member -NotePropertyName models -NotePropertyValue $remaining -Force
  Invoke-ModelStep '校验卸载后的模型目录' $catalogPath { Test-ModelCatalogShape $catalog }
  $files = @(
    @{Path=$catalogPath; Text=(ConvertTo-Json -InputObject $catalog -Depth 100)+"`n"; Kind='catalog'},
    @{Path=$managedPath; Text="[]`n"; Kind='managed'}
  )
  # Only a catalog that config.toml points at is in use; an unreadable config counts as in use.
  $active=$true
  try {
    $info=Get-ConfigInfo ([string](Read-OptionalText (Join-Path $TargetHome 'config.toml') '读取 Codex 配置'))
    $active=$info.Values.ContainsKey('model_catalog_json') -and [StringComparer]::OrdinalIgnoreCase.Equals((Get-ConfiguredCatalogPath $info $TargetHome),[IO.Path]::GetFullPath($catalogPath))
  } catch { $active=$true }
  Invoke-ModelStep '写入并校验卸载结果' $catalogPath { Write-ModelFiles $files $null }
  return @{Changed=$true; Active=$active; Count=$count}
}
