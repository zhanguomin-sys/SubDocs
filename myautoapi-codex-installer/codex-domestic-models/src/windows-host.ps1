function Read-PrivateText([string]$Path) {
  $item = Get-Item -LiteralPath $Path -Force -ErrorAction Stop
  if ($item.PSIsContainer -or ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw (New-ModelImportError '目标文件不是普通文件，停止导入。') }
  return [IO.File]::ReadAllText($Path, (New-Object Text.UTF8Encoding($false,$true)))
}

function Test-PrivatePath([string]$Path) {
  return $null -ne (Get-Item -LiteralPath $Path -Force -ErrorAction SilentlyContinue)
}

function Write-PrivateText([string]$Path, [string]$Text) {
  [IO.File]::WriteAllText($Path,$Text,(New-Object Text.UTF8Encoding($false,$true)))
}

function Move-StagedFile([string]$From, [string]$To) {
  if ([IO.File]::Exists($To)) { [IO.File]::Replace($From,$To,[NullString]::Value) }
  else { [IO.File]::Move($From,$To) }
}

function Read-ModelInput($ReadSet, [string]$Path, [string]$Step) {
  $full=[IO.Path]::GetFullPath($Path)
  foreach ($entry in $ReadSet) { if ([StringComparer]::OrdinalIgnoreCase.Equals($entry.Path,$full)) { return $entry.Text } }
  $text=Invoke-ModelStep $Step $full { if (Test-PrivatePath $full) { Read-PrivateText $full } else { $null } }
  $ReadSet.Add(@{Path=$full; Text=$text})
  return $text
}

function Test-ModelReadSet($ReadSet) {
  foreach ($entry in $ReadSet) {
    Invoke-ModelStep '核对导入期间的输入文件' $entry.Path {
      $now=if (Test-PrivatePath $entry.Path) { Read-PrivateText $entry.Path } else { $null }
      if (($null -eq $now) -ne ($null -eq $entry.Text) -or $now -cne $entry.Text) { throw (New-ModelImportError '配置或模型目录在导入过程中被修改，请重新运行。') }
    }
  }
}

function New-ModelBackendError([string]$Stderr, [int]$ExitStatus) {
  $reason='Codex 后端命令执行失败。'; $configSyntax=$false
  $toml=[regex]::Match($Stderr,'TOML parse error at line (\d+), column (\d+)','IgnoreCase')
  if (-not $toml.Success) { $toml=[regex]::Match($Stderr,'config\.toml:(\d+):(\d+):','IgnoreCase') }
  if ($toml.Success) {
    $reason='Codex 报告 TOML 配置语法错误：第 '+$toml.Groups[1].Value+' 行，第 '+$toml.Groups[2].Value+' 列。'
    $configSyntax=$Stderr -match 'TOML parse error|duplicate key|invalid table header|unterminated|expected a newline|expected newline'
  }
  elseif ($Stderr -match 'missing field [`'']([a-z_][a-z0-9_]{0,80})[`'']') {
    $known=@('slug','display_name','description','supported_reasoning_levels','shell_type','visibility','supported_in_api','priority','support_verbosity','default_verbosity','apply_patch_tool_type','truncation_policy','experimental_supported_tools','base_instructions','model_messages','instructions_template','supports_parallel_tool_calls')
    $reason=if ($known -ccontains $Matches[1]) { 'Codex 报告模型目录缺少字段：'+$Matches[1]+'。' } else { 'Codex 报告模型目录缺少必需字段。' }
  }
  elseif ($Stderr -match 'Model provider [`''][^\r\n]*[`''] not found') { $reason='model_provider 指向的服务名称未在 model_providers 中定义，请核对名称和大小写。' }
  elseif ($Stderr -match 'unrecognized subcommand|unknown subcommand|unexpected argument|unknown option') { $reason='Codex 后端未识别模型目录导出命令。' }
  elseif ($Stderr -match 'unknown variant|invalid value.*expected') { $reason='模型目录含有当前 Codex 无法识别的字段值。' }
  elseif ($Stderr -match 'failed to read model[_ ]instructions[_ ]file') { $reason='Codex 无法读取配置中的 model_instructions_file。' }
  elseif ($Stderr -match 'failed to (?:read|parse) model_catalog_json') { $reason='Codex 无法读取或解析 model_catalog_json。' }
  elseif ($Stderr -match '\b401\b|unauthorized|invalid api key') { $reason='Codex 后端请求被拒绝：鉴权失败。' }
  elseif ($Stderr -match 'connection refused|failed to connect|error sending request') { $reason='Codex 后端连接接口失败。' }
  elseif ($Stderr -match 'os error (\d+)') { $reason='Codex 后端报告系统错误，os error '+$Matches[1]+'。' }
  $problem=New-ModelImportError $reason
  $problem.Data['ModelImportExitStatus']=$ExitStatus
  if ($configSyntax) { $problem.Data['ConfigSyntax']=$true }
  return $problem
}

function Invoke-CodexExport([string]$Backend, [string[]]$Arguments, [string]$TargetHome) {
  Invoke-ModelStep '运行 Codex 模型目录命令' ($Backend+' '+($Arguments -join ' ')) {
  $start = New-Object Diagnostics.ProcessStartInfo
  $start.FileName=$Backend; $start.Arguments=$Arguments -join ' '
  $start.WorkingDirectory=$TargetHome; $start.UseShellExecute=$false; $start.CreateNoWindow=$true
  $start.RedirectStandardOutput=$true; $start.RedirectStandardError=$true
  $start.StandardOutputEncoding=New-Object Text.UTF8Encoding($false,$true)
  $start.StandardErrorEncoding=New-Object Text.UTF8Encoding($false,$true)
  $start.EnvironmentVariables['CODEX_HOME']=$TargetHome
  $process = New-Object Diagnostics.Process
  $process.StartInfo=$start
  try {
    if (-not $process.Start()) { throw (New-ModelImportError '无法启动 App 自带的 Codex 后端。') }
    $outputTask=$process.StandardOutput.ReadToEndAsync(); $errorTask=$process.StandardError.ReadToEndAsync()
    if (-not $process.WaitForExit(30000)) { $process.Kill(); $process.WaitForExit(); throw (New-ModelImportError 'App 后端校验超时，原配置未更改。') }
    $output=$outputTask.GetAwaiter().GetResult(); $stderr=$errorTask.GetAwaiter().GetResult()
    if ($process.ExitCode -ne 0) { throw (New-ModelBackendError $stderr $process.ExitCode) }
    return $output
  } finally { $process.Dispose() }
  }
}

function Write-ModelTransaction($Files, [string]$Backend, [string]$TargetHome, $Original, $ReadSet=@()) {
  $id=(Get-Date -Format 'yyyyMMddHHmmss')+'.'+[Guid]::NewGuid().ToString('N')
  $stage=Join-Path $TargetHome ('.myautoapi-models-'+$id)
  $snapshots=New-Object 'System.Collections.Generic.List[object]'
  $committed=New-Object 'System.Collections.Generic.List[int]'
  $written=New-Object 'System.Collections.Generic.List[int]'
  Test-ModelReadSet $ReadSet
  foreach ($file in $Files) {
    $snapshot=Invoke-ModelStep '检查并读取写入前的原文件' $file.Path {
    if (Test-PrivatePath $file.Path) {
      $content=Read-PrivateText $file.Path
      # Snapshot bytes as well as text, so rollback retains the original BOM/encoding exactly.
      @{Text=$content; Bytes=[IO.File]::ReadAllBytes($file.Path)}
    } else { $null }
    }
    $snapshots.Add($snapshot)
  }
  $originalNow=if ($null -ne $snapshots[2]) { $snapshots[2].Text } else { $null }
  if (($null -eq $originalNow) -ne ($null -eq $Original) -or $originalNow -cne $Original) { throw (New-ModelImportError '配置在导入过程中被修改，请重新运行。') }
  Invoke-ModelStep '创建私有暂存目录' $stage { $null=[IO.Directory]::CreateDirectory($stage) }
  try {
    for ($index=0; $index -lt $Files.Count; $index++) {
      if ($Files[$index].Keep) { continue }
      Invoke-ModelStep '写入暂存文件' (Join-Path $stage ('file-'+$index)) { Write-PrivateText (Join-Path $stage ('file-'+$index)) $Files[$index].Text }
    }
    $stagedCatalog=Join-Path $stage 'file-0'
    $levels=@()
    if ($Backend) {
      $info=Get-ConfigInfo ([string]$Files[2].Text)
      if ($info.Values.ContainsKey('desktop.enabled-reasoning-efforts')) { $levels=Get-TomlStringArray $info.Values['desktop.enabled-reasoning-efforts'] }
    }
    $validationConfig='model_catalog_json = '+(ConvertTo-Json -InputObject $stagedCatalog -Compress)+"`n[desktop]`nenabled-reasoning-efforts = "+(ConvertTo-Json -InputObject @($levels) -Compress)+"`n"
    Write-PrivateText (Join-Path $stage 'config.toml') $validationConfig
    $validated=if ($Backend) { ConvertFrom-ModelJson (Invoke-CodexExport $Backend @('debug','models') $stage) '解析暂存模型目录校验结果' ($Backend+' debug models') }
      else { ConvertFrom-ModelJson (Read-PrivateText $stagedCatalog) '解析暂存模型目录校验结果' $stagedCatalog }
    Test-ModelCatalog $validated
    $expected=ConvertFrom-Json -InputObject $Files[0].Text
    $actualIds=New-OrdinalMap; foreach ($model in $validated.models) { $actualIds[$model.slug]=$true }
    if ($actualIds.Count -ne $expected.models.Count) { throw (New-ModelImportError 'App 未读取暂存模型目录，停止导入。') }
    foreach ($model in $expected.models) { if (-not $actualIds.ContainsKey($model.slug)) { throw (New-ModelImportError 'App 未读取暂存模型目录，停止导入。') } }
    Test-ModelReadSet $ReadSet
    for ($index=0; $index -lt $Files.Count; $index++) {
      $now=if (Test-PrivatePath $Files[$index].Path) { Read-PrivateText $Files[$index].Path } else { $null }
      $before=if ($null -ne $snapshots[$index]) { $snapshots[$index].Text } else { $null }
      if (($null -eq $now) -ne ($null -eq $before) -or $now -cne $before) { throw (New-ModelImportError '配置在导入过程中被修改，请重新运行。') }
    }
    for ($index=0; $index -lt $Files.Count; $index++) {
      if ($Files[$index].Keep) { continue }
      if ($null -ne $snapshots[$index]) { Invoke-ModelStep '备份原文件' $Files[$index].Path { [IO.File]::Copy($Files[$index].Path,($Files[$index].Path+'.bak.'+$id),$false) } }
    }
    for ($index=0; $index -lt $Files.Count; $index++) {
      if ($Files[$index].Keep) { continue }
      $committed.Add($index)
      Invoke-ModelStep '提交导入文件' $Files[$index].Path { Move-StagedFile (Join-Path $stage ('file-'+$index)) $Files[$index].Path }
      $written.Add($index)
    }
    # Validate the committed catalog without loading or changing the user's provider/auth.
    $validationConfig='model_catalog_json = '+(ConvertTo-Json -InputObject $Files[0].Path -Compress)+"`n[desktop]`nenabled-reasoning-efforts = "+(ConvertTo-Json -InputObject @($levels) -Compress)+"`n"
    Write-PrivateText (Join-Path $stage 'config.toml') $validationConfig
    $final=if ($Backend) { ConvertFrom-ModelJson (Invoke-CodexExport $Backend @('debug','models') $stage) '解析已写入模型目录校验结果' ($Backend+' debug models') }
      else { ConvertFrom-ModelJson (Read-PrivateText $Files[0].Path) '解析已写入模型目录校验结果' $Files[0].Path }
    Test-ModelCatalog $final
    $finalIds=New-OrdinalMap; foreach ($model in $final.models) { $finalIds[$model.slug]=$true }
    if ($finalIds.Count -ne $expected.models.Count) { throw (New-ModelImportError 'App 未读取新模型目录，停止导入。') }
    foreach ($model in $expected.models) { if (-not $finalIds.ContainsKey($model.slug)) { throw (New-ModelImportError 'App 未读取新模型目录，停止导入。') } }
    $normalized=ConvertTo-Json -InputObject (ConvertTo-CanonicalValue $validated.models) -Depth 100 -Compress
    $effective=ConvertTo-Json -InputObject (ConvertTo-CanonicalValue $final.models) -Depth 100 -Compress
    if ($normalized -cne $effective) { throw (New-ModelImportError '已写入的模型元数据与本次导入不一致，停止导入。') }
    foreach ($file in $Files) {
      $now=if (Test-PrivatePath $file.Path) { Read-PrivateText $file.Path } else { $null }
      if (($null -eq $now) -ne ($null -eq $file.Text) -or $now -cne $file.Text) { throw (New-ModelImportError '写入后的文件与本次导入不一致，停止导入。') }
    }
  } catch {
    if ($committed.Count) {
      $conflictPath=$null
      for ($index=0; $index -lt $Files.Count; $index++) {
        $now=$null; $found=$false
        try {
          $null=Get-Item -LiteralPath $Files[$index].Path -Force -ErrorAction Stop
          $found=$true
          $now=Read-PrivateText $Files[$index].Path
        } catch {
          if ($found -or $_.CategoryInfo.Category -ne [System.Management.Automation.ErrorCategory]::ObjectNotFound) { $conflictPath=$Files[$index].Path; break }
        }
        $matchesOutput=$null -ne $now -and $now -ceq $Files[$index].Text
        $matchesBefore=if ($null -eq $snapshots[$index]) { $null -eq $now } else { $null -ne $now -and $now -ceq $snapshots[$index].Text }
        if (-not $matchesOutput -and ($written.Contains($index) -or -not $matchesBefore)) { $conflictPath=$Files[$index].Path; break }
      }
      if ($conflictPath) {
        $problem=New-ModelImportError '导入过程中出现外部修改，已保留当前内容和备份，软件未关闭。'
        $problem.Data['ModelImportStep']='核对失败后的文件状态'; $problem.Data['ModelImportTarget']=$conflictPath
        throw $problem
      }
    }
    $restoreFailed=$false
    for ($step=$committed.Count-1; $step -ge 0; $step--) {
      $index=$committed[$step]
      try {
        if ($null -eq $snapshots[$index]) { [IO.File]::Delete($Files[$index].Path) }
        else {
          $restorePath=Join-Path $stage ('restore-'+$index)
          [IO.File]::WriteAllBytes($restorePath,$snapshots[$index].Bytes)
          Move-StagedFile $restorePath $Files[$index].Path
        }
      } catch { $restoreFailed=$true }
    }
    if ($restoreFailed) { throw (New-ModelImportError '写入失败且恢复未完成，请使用同目录的 .bak 备份恢复；软件未关闭。') }
    throw
  } finally {
    if ([IO.Directory]::Exists($stage)) { try { [IO.Directory]::Delete($stage,$true) } catch { } }
  }
}

function Find-DesktopBackend([string]$ExplicitPath) {
  if ($ExplicitPath) {
    $full=[IO.Path]::GetFullPath($ExplicitPath)
    if (-not [IO.File]::Exists($full)) { throw (New-ModelImportError '指定的 App 后端文件不存在。') }
    return $full
  }
  $roots=New-Object 'System.Collections.Generic.List[string]'
  if (Get-Command Get-AppxPackage -ErrorAction SilentlyContinue) {
    foreach ($package in @(Get-AppxPackage -Name 'OpenAI.Codex' -ErrorAction SilentlyContinue)) {
      if ($package.InstallLocation) { $roots.Add($package.InstallLocation); $roots.Add((Join-Path $package.InstallLocation 'app')) }
    }
  }
  foreach ($base in @($env:LOCALAPPDATA,$env:ProgramFiles)) {
    if (-not $base) { continue }
    foreach ($relative in @('Programs\Codex','Programs\ChatGPT','Codex','ChatGPT')) { $roots.Add((Join-Path $base $relative)) }
  }
  foreach ($root in $roots) {
    foreach ($relative in @('resources\codex.exe','resources\bin\codex.exe')) {
      $candidate=Join-Path $root $relative
      if ([IO.File]::Exists($candidate)) { return $candidate }
    }
  }
  throw (New-ModelImportError '未找到桌面 App 自带的 Codex 后端；用 -CodexPath 指定其 resources 中的 codex.exe。')
}

function Start-ModelInstaller([string]$CodexHomePath, [string]$CodexPath, [switch]$Uninstall) {
  if ($env:OS -ne 'Windows_NT') { throw (New-ModelImportError '这个脚本用于 Windows；macOS 请运行 codex-models.sh。') }
  if (-not $CodexHomePath) { $CodexHomePath=if ($env:CODEX_HOME) { $env:CODEX_HOME } else { Join-Path $HOME '.codex' } }
  $targetHome=[IO.Path]::GetFullPath($CodexHomePath)
  if ($Uninstall) {
    Write-Host '移除当前启用目录中的国产模型后会自动关闭 ChatGPT/Codex，运行中的任务会中断。'
    $result=Remove-ImportedModels $targetHome
    if (-not $result.Changed) { Write-Host '没有需要移除的国产模型，软件未关闭。'; return }
    if (-not $result.Active) { Write-Host '已清理原导入目录；软件未关闭。'; return }
    Write-Host '已移除本工具导入的国产模型，原 GPT 和其他模型已保留。'
    Report-RunningClients (Stop-RunningClients)
    Write-Host '重新打开 App，新建任务后选择模型。'
    return
  }
  Write-Host '成功导入后会自动关闭 ChatGPT/Codex，运行中的任务会中断。'
  $backend=Find-DesktopBackend $CodexPath
  $count=Invoke-ModelImport $targetHome $backend $DomesticCatalog
  Write-Host "已导入 $count 个国产模型，原 GPT 和其他模型已保留。"
  Report-RunningClients (Stop-RunningClients)
  Write-Host '重新打开 App，新建任务后选择模型；API Key 分组由你在中转站切换。'
}
