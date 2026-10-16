function Read-PrivateText([string]$Path) {
  $item = Get-Item -LiteralPath $Path -Force -ErrorAction Stop
  if ($item.PSIsContainer -or ($item.Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw (New-ModelImportError '目标文件不是普通文件，停止导入。') }
  return [IO.File]::ReadAllText($Path, (New-Object Text.UTF8Encoding($false,$true)))
}

function Test-PrivatePath([string]$Path) {
  return $null -ne (Get-Item -LiteralPath $Path -Force -ErrorAction SilentlyContinue)
}

function Test-Utf8Bom([string]$Path) {
  if (-not (Test-PrivatePath $Path)) { return $false }
  $head = New-Object byte[] 3
  $stream = [IO.File]::OpenRead($Path)
  try { $count = $stream.Read($head,0,3) } finally { $stream.Dispose() }
  return ($count -eq 3 -and $head[0] -eq 0xEF -and $head[1] -eq 0xBB -and $head[2] -eq 0xBF)
}

function Write-PrivateText([string]$Path, [string]$Text, [bool]$Bom=$false) {
  [IO.File]::WriteAllText($Path,$Text,(New-Object Text.UTF8Encoding($Bom,$true)))
}

function Move-StagedFile([string]$From, [string]$To) {
  if ([IO.File]::Exists($To)) { [IO.File]::Replace($From,$To,[NullString]::Value) }
  else { [IO.File]::Move($From,$To) }
}

function Read-OptionalText([string]$Path, [string]$Step) {
  return (Invoke-ModelStep $Step $Path { if (Test-PrivatePath $Path) { Read-PrivateText $Path } else { $null } })
}

# Writes the catalog, ownership record and config. Each file is backed up, written
# to a temporary file next to it, checked, then moved into place; config.toml goes
# last so it never points at a catalog that was not written. There is no rollback:
# if a later move fails, the earlier files are already replaced and every original
# stays next to it as .bak.<id>.
function Write-ModelFiles($Files, $OriginalConfig) {
  $id = (Get-Date -Format 'yyyyMMddHHmmss') + '.' + [Guid]::NewGuid().ToString('N')
  $staged = New-Object 'System.Collections.Generic.List[string]'
  try {
    foreach ($file in $Files) {
      $file.Temp = $file.Path + '.tmp.' + $id
      $file.Bom = Test-Utf8Bom $file.Path
      $staged.Add($file.Temp)
      Invoke-ModelStep '写入暂存文件' $file.Temp { Write-PrivateText $file.Temp $file.Text $file.Bom }
      $written = Read-PrivateText $file.Temp
      Invoke-ModelStep '校验暂存文件' $file.Path {
        if ($file.Kind -ceq 'catalog') { Test-ModelCatalogShape (ConvertFrom-ModelJson $written '解析暂存模型目录' $file.Path) }
        elseif ($file.Kind -ceq 'config') { $null = Get-ConfigInfo $written }
        else { $null = ConvertFrom-ModelJson $written '解析暂存管理记录' $file.Path }
      }
    }
    foreach ($file in $Files) {
      if ($file.Kind -ceq 'config') {
        $now = if (Test-PrivatePath $file.Path) { Read-PrivateText $file.Path } else { $null }
        if (($null -eq $now) -ne ($null -eq $OriginalConfig) -or $now -cne $OriginalConfig) { throw (New-ModelImportError '配置在导入过程中被修改，请重新运行。') }
      }
      if (Test-PrivatePath $file.Path) { Invoke-ModelStep '备份原文件' $file.Path { [IO.File]::Copy($file.Path,($file.Path+'.bak.'+$id),$false) } }
    }
    foreach ($file in $Files) {
      Invoke-ModelStep '提交导入文件' $file.Path { Move-StagedFile $file.Temp $file.Path }
    }
  } finally {
    foreach ($path in $staged) { if ([IO.File]::Exists($path)) { try { [IO.File]::Delete($path) } catch { } } }
  }
}

function Start-ModelInstaller([string]$CodexHomePath, [string]$CodexPath, [switch]$Uninstall) {
  if ($env:OS -ne 'Windows_NT') { throw (New-ModelImportError '这个脚本用于 Windows；macOS 请运行 codex-models.sh。') }
  if (-not $CodexHomePath) { $CodexHomePath=if ($env:CODEX_HOME) { $env:CODEX_HOME } else { Join-Path $HOME '.codex' } }
  $targetHome=[IO.Path]::GetFullPath($CodexHomePath)
  # CodexPath remains accepted for compatibility with saved commands, but is
  # deliberately unused. Import and uninstall never locate or start Codex; they only
  # read and write files, then close running clients with the one-click installer's code.
  if ($Uninstall) {
    Write-Host '移除当前启用目录中的国产模型后会自动关闭 ChatGPT/Codex，运行中的任务会中断。'
    $result=Remove-ImportedModels $targetHome $DomesticCatalog
    if (-not $result.Changed) { Write-Host '没有需要移除的国产模型，软件未关闭。'; return }
    if (-not $result.Active) { Write-Host '已清理原导入目录；软件未关闭。'; return }
    Write-Host '已移除本工具导入的国产模型，原 GPT 和其他模型已保留。'
    Report-RunningClients (Stop-RunningClients)
    Write-Host '重新打开 App，新建任务后选择模型。'
    return
  }
  Write-Host '成功导入后会自动关闭 ChatGPT/Codex，运行中的任务会中断。'
  $result=Invoke-ModelImport $targetHome $DomesticCatalog
  Write-Host ("已导入 "+$result.Count+" 个国产模型，原 GPT 和其他模型已保留。")
  if (-not $result.UsedCache) { Write-Host '未读到本机模型缓存，GPT 列表来自脚本内置快照；Codex 更新后如缺少新模型，重新运行本命令即可。' }
  Report-RunningClients (Stop-RunningClients)
  Write-Host '重新打开 App，新建任务后选择模型；API Key 分组由你在中转站切换。'
}
