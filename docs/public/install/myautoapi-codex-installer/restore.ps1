$ErrorActionPreference = "Stop"

$CodexHome = Join-Path $HOME ".codex"
$ConfigFile = Join-Path $CodexHome "config.toml"
$AuthFile = Join-Path $CodexHome "auth.json"

function Backup-IfExists([string]$Path) {
  if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
    return
  }

  $stamp = Get-Date -Format "yyyyMMddHHmmss"
  $backup = "$Path.bak.$stamp"
  $suffix = 1
  while (Test-Path -LiteralPath $backup) {
    $backup = "$Path.bak.$stamp.$suffix"
    $suffix += 1
  }

  Copy-Item -LiteralPath $Path -Destination $backup
}

function Write-Utf8NoBom([string]$Path, [string]$Content) {
  $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
  [System.IO.File]::WriteAllText($Path, $Content, $utf8NoBom)
}

# Track TOML multi-line strings so their contents are never read as tables or keys.
function Update-MultilineState([string]$Line, [string]$Open) {
  $rest = $Line
  while ($true) {
    if ($Open -eq '') {
      $doubleAt = $rest.IndexOf('"""')
      $singleAt = $rest.IndexOf("'''")
      if ($doubleAt -lt 0 -and $singleAt -lt 0) {
        return $Open
      }
      if ($doubleAt -ge 0 -and ($singleAt -lt 0 -or $doubleAt -lt $singleAt)) {
        $Open = '"""'
        $at = $doubleAt
      } else {
        $Open = "'''"
        $at = $singleAt
      }
    } else {
      $at = $rest.IndexOf($Open)
      if ($at -lt 0) {
        return $Open
      }
      $Open = ''
    }
    $rest = $rest.Substring($at + 3)
  }
}

function Clear-CodexConfig {
  if (-not (Test-Path -LiteralPath $ConfigFile -PathType Leaf)) {
    return
  }

  Backup-IfExists $ConfigFile
  $content = [System.IO.File]::ReadAllText($ConfigFile)
  $newline = if ($content.Contains("`r`n")) { "`r`n" } else { "`n" }
  $lines = [System.Text.RegularExpressions.Regex]::Split($content, "`r`n|`n|`r")
  $keptLines = New-Object System.Collections.Generic.List[string]
  $inRoot = $true
  $skipProvider = $false
  $removedRoot = $false
  $multiline = ''
  $skipValue = $false

  foreach ($line in $lines) {
    $inString = $multiline -ne ''
    $multiline = Update-MultilineState $line $multiline
    if ($inString) {
      if ($skipProvider -or $skipValue) {
        if ($multiline -eq '') {
          $skipValue = $false
        }
        continue
      }
      $keptLines.Add($line)
      continue
    }

    $isTable = $line -match '^[\t ]*\[\[?[^\]]+\]\]?[\t ]*(?:#.*)?$'
    $isProvider = $line -match '^[\t ]*\[model_providers\.[^\]]+\][\t ]*(?:#.*)?$'

    if ($isTable) {
      $inRoot = $false
      if ($isProvider) {
        $skipProvider = $true
        continue
      }
      $skipProvider = $false
    }

    if ($skipProvider) {
      continue
    }

    if ($inRoot -and $line -match '^[\t ]*(?:model_provider|model|review_model)[\t ]*=') {
      $removedRoot = $true
      $skipValue = $multiline -ne ''
      continue
    }

    if ($removedRoot -and $keptLines.Count -eq 0 -and [string]::IsNullOrWhiteSpace($line)) {
      continue
    }

    $keptLines.Add($line)
  }

  $configTemp = Join-Path $CodexHome (".config.toml.restore." + [Guid]::NewGuid().ToString("N"))
  try {
    Write-Utf8NoBom -Path $configTemp -Content ([string]::Join($newline, $keptLines))
    Move-Item -LiteralPath $configTemp -Destination $ConfigFile -Force
  } finally {
    Remove-Item -LiteralPath $configTemp -Force -ErrorAction SilentlyContinue
  }
}

function Clear-CodexAuth {
  if (-not (Test-Path -LiteralPath $AuthFile -PathType Leaf)) {
    return
  }

  Backup-IfExists $AuthFile
  [System.IO.File]::WriteAllBytes($AuthFile, [byte[]]@())
}

if (-not (Test-Path -LiteralPath $CodexHome -PathType Container)) {
  Write-Host "未找到 Codex 配置，无需还原。"
  return
}

Clear-CodexConfig
Clear-CodexAuth

Write-Host "Codex 自定义模型配置和登录凭据已清除。"
Write-Host "请彻底退出 Codex，重新打开后使用官方账号登录。"
