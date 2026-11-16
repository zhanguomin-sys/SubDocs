$ErrorActionPreference = "Stop"

$CodexBaseUrl = "https://api.myautoapi.com"
$CodexModel = "gpt-6.1-sol"
$UsageUrl = "$CodexBaseUrl/v1/usage"

$CodexHome = Join-Path $HOME ".codex"
$ConfigFile = Join-Path $CodexHome "config.toml"
$AuthFile = Join-Path $CodexHome "auth.json"

function Say([string]$Message) {
  Write-Host $Message
  if (-not [string]::IsNullOrEmpty($Message)) {
    Start-Sleep -Milliseconds 500
  }
}

function Backup-IfExists([string]$Path) {
  if (Test-Path -LiteralPath $Path) {
    $stamp = Get-Date -Format "yyyyMMddHHmmss"
    $backup = "$Path.bak.$stamp"
    $suffix = 1
    while (Test-Path -LiteralPath $backup) {
      $backup = "$Path.bak.$stamp.$suffix"
      $suffix += 1
    }
    Copy-Item -LiteralPath $Path -Destination $backup
  }
}

function Get-RunningCodexClients {
  $sessionId = (Get-Process -Id $PID).SessionId
  return @(
    Get-Process -Name "codex", "chatgpt" -ErrorAction SilentlyContinue |
      Where-Object { $_.SessionId -eq $sessionId }
  )
}

function Get-OwnProcessIds {
  $ids = @()
  $current = $PID
  while ($current -and ($ids -notcontains $current)) {
    $ids += $current
    $process = Get-CimInstance Win32_Process -Filter "ProcessId = $current" -ErrorAction Stop
    if ($process) {
      $current = $process.ParentProcessId
    } else {
      $current = 0
    }
  }
  return $ids
}

function Test-ProcessExited($Process) {
  try {
    return $Process.HasExited
  } catch {
    return $false
  }
}

function Stop-RunningClients {
  $result = @{ ChatGPT = $false; Codex = $false; Left = $false }
  try {
    $clients = @(Get-RunningCodexClients)
    $result.ChatGPT = @($clients | Where-Object { $_.ProcessName -ieq "chatgpt" }).Count -gt 0
    $result.Codex = @($clients | Where-Object { $_.ProcessName -ieq "codex" }).Count -gt 0
    if ($clients.Count -eq 0) {
      return $result
    }

    $ownIds = @(Get-OwnProcessIds)
    $targets = @($clients | Where-Object { $ownIds -notcontains $_.Id })
    if ($targets.Count -lt $clients.Count) {
      $result.Left = $true
    }
    if ($targets.Count -gt 0) {
      $targets | Stop-Process -Force -ErrorAction SilentlyContinue
      for ($attempt = 0; $attempt -le 10; $attempt++) {
        if ($attempt -gt 0) {
          Start-Sleep -Milliseconds 500
        }
        $remaining = @($targets | Where-Object { -not (Test-ProcessExited $_) })
        if ($remaining.Count -eq 0) {
          break
        }
      }
      if ($remaining.Count -gt 0) {
        $result.Left = $true
      }
    }
  } catch {
    $result.Left = $true
  }
  return $result
}

function Report-RunningClients($Result) {
  if ($Result.ChatGPT -and $Result.Codex) {
    $clients = "ChatGPT 软件和 Codex"
  } elseif ($Result.ChatGPT) {
    $clients = "ChatGPT 软件"
  } elseif ($Result.Codex) {
    $clients = "Codex"
  } elseif ($Result.Left) {
    $clients = "ChatGPT 软件和 Codex"
  } else {
    Say "当前未检测到 ChatGPT/Codex CLI 运行，请启动软件后进行测试。"
    return
  }

  if ($Result.Left) {
    Say "未能自动关闭 ${clients}，请重启电脑后，重新打开并新建对话测试，如有问题请联系网站客服。"
    return
  }
  Say "已自动关闭 ${clients}，请重新打开并新建对话测试，如有问题请联系网站客服。"
}

function Show-WritePlan {
  $existingFiles = @()
  foreach ($path in @($ConfigFile, $AuthFile)) {
    if (Test-Path -LiteralPath $path -PathType Leaf) {
      $existingFiles += $path
    }
  }

  if ($existingFiles.Count -eq 0) {
    return
  }

  Say ""
  Say "检测到已有配置，会先备份；config.toml 改写中转站接口和模型，去掉以前写入的完全访问设置和其它接口，其余设置保留。"
}

function Read-ValidatedApiKey {
  while ($true) {
    $apiKey = Read-Host "请粘贴从 MyAutoAPI 中转站复制过来的 API Key，然后回车确认"
    if ([string]::IsNullOrWhiteSpace($apiKey)) {
      Say ""
      Say "API Key 不能为空，请重新输入。"
      Say ""
      continue
    }
    if (-not $apiKey.StartsWith("sk-", [System.StringComparison]::Ordinal)) {
      Say ""
      Say "API Key 格式错误，必须以 sk- 开头，请重新输入。"
      Say ""
      continue
    }
    return $apiKey
  }
}

# 检查结果 State：usable、no_balance、rejected、disabled、unknown。
# 按 JSON 规则转义字符串：引号、反斜杠和控制字符。
function ConvertTo-JsonText([string]$Value) {
  $builder = New-Object System.Text.StringBuilder
  foreach ($character in $Value.ToCharArray()) {
    $code = [int]$character
    if ($code -eq 0x22) {
      [void]$builder.Append('\"')
    } elseif ($code -eq 0x5C) {
      [void]$builder.Append('\\')
    } elseif ($code -lt 0x20) {
      [void]$builder.Append(('\u{0:x4}' -f $code))
    } else {
      [void]$builder.Append($character)
    }
  }
  return $builder.ToString()
}

# /v1/usage 不做计费检查：过期或额度用完的密钥仍返回 isValid true，要看 status 和 expires_at。
# 到期后 status 仍是 active，所以还要比较过期时间。
function Test-KeyEnded($Data) {
  if ($Data.status -eq "expired" -or $Data.status -eq "quota_exhausted") {
    return $true
  }
  $expires = $Data.expires_at
  if ($expires -is [DateTimeOffset]) {
    return $expires.UtcDateTime -lt [DateTime]::UtcNow
  }
  if ($expires -is [DateTime]) {
    return $expires.ToUniversalTime() -lt [DateTime]::UtcNow
  }
  if ($expires -is [string] -and $expires) {
    $parsed = [DateTimeOffset]::MinValue
    if ([DateTimeOffset]::TryParse($expires, [System.Globalization.CultureInfo]::InvariantCulture, [System.Globalization.DateTimeStyles]::AssumeUniversal, [ref]$parsed)) {
      return $parsed.UtcDateTime -lt [DateTime]::UtcNow
    }
  }
  return $false
}

# 超时的样子：Windows PowerShell 5.1 是 Status 为 Timeout 的 WebException，
# PowerShell 7 是 TaskCanceledException 或内层的 TimeoutException。
function Test-Timeout($Exception) {
  $current = $Exception
  while ($null -ne $current) {
    if ($current -is [System.TimeoutException] -or $current -is [System.OperationCanceledException]) {
      return $true
    }
    if ($current -is [System.Net.WebException] -and $current.Status -eq [System.Net.WebExceptionStatus]::Timeout) {
      return $true
    }
    $current = $current.InnerException
  }
  return $false
}

# State：usable、no_balance、rejected、disabled，检查不了时为 timeout（10 秒内没有回应）、
# offline（连不上）或 server_error（服务器出错或返回异常，Http 为状态码）。
function Test-ApiKey([string]$ApiKey) {
  $result = @{ State = "unknown"; Balance = $null; Http = $null }
  try {
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
  } catch {
  }
  try {
    $response = Invoke-WebRequest -Uri $UsageUrl -Headers @{ Authorization = "Bearer $ApiKey"; Accept = "application/json" } -TimeoutSec 10 -UseBasicParsing -ErrorAction Stop
  } catch {
    $status = 0
    try {
      $status = [int]$_.Exception.Response.StatusCode
    } catch {
    }
    if ($status -eq 401 -or $status -eq 403) {
      $result.State = "rejected"
    } elseif ($status -gt 0) {
      $result.State = "server_error"
      $result.Http = $status
    } elseif (Test-Timeout $_.Exception) {
      $result.State = "timeout"
    } else {
      $result.State = "offline"
    }
    return $result
  }
  if ([int]$response.StatusCode -ne 200) {
    $result.State = "server_error"
    $result.Http = [int]$response.StatusCode
    return $result
  }
  try {
    $data = $response.Content | ConvertFrom-Json -ErrorAction Stop
  } catch {
    $result.State = "server_error"
    return $result
  }
  if (-not ($data.isValid -is [bool])) {
    $result.State = "server_error"
    return $result
  }
  if (-not $data.isValid -or (Test-KeyEnded $data)) {
    $result.State = "disabled"
    return $result
  }
  $result.State = "usable"
  if ($null -ne $data.remaining) {
    try {
      $remaining = [double]$data.remaining
    } catch {
      return $result
    }
    $result.Balance = $remaining.ToString("0.00", [System.Globalization.CultureInfo]::InvariantCulture)
    if ($remaining -le 0) {
      $result.State = "no_balance"
    }
  }
  return $result
}

function Get-CheckedApiKey {
  $apiKey = $null
  $fromEnvironment = $env:MYAUTOAPI_KEY
  Remove-Item -LiteralPath Env:\MYAUTOAPI_KEY -ErrorAction SilentlyContinue
  if (-not [string]::IsNullOrEmpty($fromEnvironment)) {
    if ($fromEnvironment.StartsWith("sk-", [System.StringComparison]::Ordinal)) {
      $apiKey = $fromEnvironment
    } else {
      Say "MYAUTOAPI_KEY 格式错误，必须以 sk- 开头，请手动输入。"
      Say ""
    }
  }

  while ($true) {
    if (-not $apiKey) {
      $apiKey = Read-ValidatedApiKey
    }
    $check = Test-ApiKey $apiKey
    if ($check.State -eq "rejected") {
      Say ""
      Say "密钥无效，请回网站重新复制完整的密钥。"
      Say ""
      $apiKey = $null
      continue
    }
    if ($check.State -eq "disabled") {
      Say ""
      Say "这个密钥已停用、过期或额度用完，请在网站换一个可用的密钥。"
      Say ""
      $apiKey = $null
      continue
    }
    Say ""
    if ($check.State -eq "no_balance") {
      Say "密钥可用，但余额为 `$0。充值后即可使用；新人进群领 2 元。"
    } elseif ($check.State -eq "timeout") {
      Say "网络超时：MyAutoAPI 服务器 10 秒内没有回应，暂时无法检查密钥，继续写入。"
    } elseif ($check.State -eq "offline") {
      Say "连不上 MyAutoAPI 服务器（网络不通，或被代理、防火墙拦截），暂时无法检查密钥，继续写入。"
    } elseif ($check.State -eq "server_error") {
      if ($null -ne $check.Http) {
        Say ("MyAutoAPI 服务器暂时出错（HTTP " + $check.Http + "），暂时无法检查密钥，继续写入。")
      } else {
        Say "MyAutoAPI 服务器暂时出错（返回内容异常），暂时无法检查密钥，继续写入。"
      }
    } elseif ($check.State -eq "unknown") {
      Say "暂时无法检查密钥，继续写入。"
    } elseif ($null -ne $check.Balance) {
      Say ("密钥检查通过，余额 `$" + $check.Balance + "。")
    } else {
      Say "密钥检查通过。"
    }
    return $apiKey
  }
}

function Write-Utf8NoBom([string]$Path, [string]$Content) {
  $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
  [System.IO.File]::WriteAllText($Path, $Content, $utf8NoBom)
}

# 跟踪 TOML 多行字符串和数组、行内表的括号，字符串里的内容不会被当成表或键。返回这一行
# 结束时还没关上的 """ 或 '''，以及 $Depth 加上这一行打开减去关上的 [ { 个数；注释和单行
# 字符串整段跳过，里面的引号和括号不算。与 connect/codexfiles.go 的 scanLine 相同
# （codex-restore.ps1 仍用旧的写法）。
function Update-MultilineState([string]$Line, [string]$Open, [int]$Depth) {
  $i = 0
  while ($i -lt $Line.Length) {
    $c = $Line[$i]
    if ($Open -ne '') {
      if ($Open -eq '"""' -and $c -eq [char]'\') {
        $i += 2
      } elseif ([string]::CompareOrdinal($Line, $i, $Open, 0, 3) -eq 0) {
        # 后面最多再两个引号也属于字符串。
        $i += 3
        $extra = 0
        while ($extra -lt 2 -and $i -lt $Line.Length -and $Line[$i] -eq $Open[0]) {
          $i++
          $extra++
        }
        $Open = ''
      } else {
        $i++
      }
    } elseif ($c -eq [char]'#') {
      return '', $Depth
    } elseif ($c -eq [char]'[' -or $c -eq [char]'{') {
      $Depth++
      $i++
    } elseif ($c -eq [char]']' -or $c -eq [char]'}') {
      $Depth--
      $i++
    } elseif ([string]::CompareOrdinal($Line, $i, '"""', 0, 3) -eq 0 -or [string]::CompareOrdinal($Line, $i, "'''", 0, 3) -eq 0) {
      $Open = $Line.Substring($i, 3)
      $i += 3
    } elseif ($c -eq [char]'"' -or $c -eq [char]"'") {
      $i++
      while ($i -lt $Line.Length -and $Line[$i] -ne $c) {
        if ($c -eq [char]'"' -and $Line[$i] -eq [char]'\') {
          $i++
        }
        $i++
      }
      $i++
    } else {
      $i++
    }
  }
  return $Open, $Depth
}

# 去掉开头和结尾只有空格、Tab 的行（TOML 只认这两种空白）。
function Remove-BlankEnds([System.Collections.Generic.List[string]]$Lines) {
  $blank = [char[]]@([char]' ', [char]9)
  while ($Lines.Count -gt 0 -and $Lines[0].Trim($blank) -eq '') { $Lines.RemoveAt(0) }
  while ($Lines.Count -gt 0 -and $Lines[$Lines.Count - 1].Trim($blank) -eq '') { $Lines.RemoveAt($Lines.Count - 1) }
}

# 导入时 config.toml 只写中转站这几行：开头的 model_provider、model，以及作为第一个段的
# [model_providers.OpenAI]。删掉的内容：开头的 review_model、以前版本写的完全访问和沙箱
# 设置、推理强度、国产模型列表 model_catalog_json（重新接入后国产模型照旧要再导入）、
# profile（会盖过 model 的设置）、会让 Key 不生效的登录设置 forced_login_method 和
# cli_auth_credentials_store；所有接口段 [model_providers.*]，包括写成
# model_providers.xx = 或 {…} 形式的。其它内容原样保留（包括 Codex App 设置沙箱时写的
# [windows]），旧文件另有备份。取舍见 quick-connect/取舍记录.md。
# 与快速接入助手 connect/codexfiles.go 的 ConfigFor 逐字节一致。
function Get-CodexConfig([string]$Old) {
  $root = "model_provider = `"OpenAI`"`nmodel = `"$CodexModel`""
  $provider = "[model_providers.OpenAI]`nname = `"OpenAI`"`nbase_url = `"$CodexBaseUrl`"`nwire_api = `"responses`"`nrequires_openai_auth = true"
  if ($Old.StartsWith([string][char]0xFEFF, [System.StringComparison]::Ordinal)) { $Old = $Old.Substring(1) }
  $newline = if ($Old.Contains("`r`n")) { "`r`n" } else { "`n" }
  # 表头按 TOML 键名写法识别：裸键、带转义的 "…"、'…'，用点连接；引号里的 ] 不算表头结束。
  $key = '(?:[A-Za-z0-9_-]+|"(?:[^"\\]|\\.)*"|''[^'']*'')'
  $headerPattern = '^[\t ]*\[\[?[\t ]*' + $key + '(?:[\t ]*\.[\t ]*' + $key + ')*[\t ]*\]\]?[\t ]*(?:#.*)?$'
  # 要删的根级键和 model_providers 都可以写成裸键、"…" 或 '…'。
  $names = '(?:model_provider|model|review_model|model_reasoning_effort|disable_response_storage|approval_policy|sandbox_mode|network_access|windows_wsl_setup_acknowledged|model_catalog_json|profile|forced_login_method|cli_auth_credentials_store)'
  $providers = '(?:model_providers|"model_providers"|''model_providers'')'
  $rootPattern = '^[\t ]*(?:(?:' + $names + '|"' + $names + '"|''' + $names + ''')[\t ]*=|' + $providers + '[\t ]*[.=])'
  $providerPattern = '^[\t ]*\[[\t ]*' + $providers + '(?:[\t ]*\.[\t ]*' + $key + ')*[\t ]*\][\t ]*(?:#.*)?$'
  $rootLines = New-Object System.Collections.Generic.List[string]
  $tableLines = New-Object System.Collections.Generic.List[string]
  $inRoot = $true
  $skipTable = $false
  $multiline = ''
  $depth = 0
  $skipValue = $false
  foreach ($line in [System.Text.RegularExpressions.Regex]::Split($Old, "`r`n|`n|`r")) {
    if ($skipValue) {
      # 被删键的值还没结束：多行字符串、数组或行内表关上之前都跳过。
      $multiline, $depth = Update-MultilineState $line $multiline $depth
      $skipValue = $multiline -ne '' -or $depth -gt 0
      continue
    }
    $inString = $multiline -ne ''
    $multiline, $depth = Update-MultilineState $line $multiline 0
    if ($inString) {
      if ($skipTable) {
        continue
      }
    } else {
      if ($line -match $headerPattern) {
        $inRoot = $false
        $skipTable = $line -match $providerPattern
      }
      if ($skipTable) {
        continue
      }
      if ($inRoot -and $line -match $rootPattern) {
        $skipValue = $multiline -ne '' -or $depth -gt 0
        continue
      }
    }
    if ($inRoot) { $rootLines.Add($line) } else { $tableLines.Add($line) }
  }
  Remove-BlankEnds $rootLines
  Remove-BlankEnds $tableLines
  $head = New-Object System.Collections.Generic.List[string]
  $head.AddRange([string[]]$root.Split("`n"))
  $head.AddRange($rootLines)
  $parts = New-Object System.Collections.Generic.List[string]
  $parts.Add([string]::Join($newline, $head))
  $parts.Add($provider.Replace("`n", $newline))
  if ($tableLines.Count -gt 0) { $parts.Add([string]::Join($newline, $tableLines)) }
  return [string]::Join($newline + $newline, $parts)
}

function Write-CodexFiles([string]$ApiKey) {
  if (-not (Test-Path -LiteralPath $CodexHome)) {
    New-Item -ItemType Directory -Path $CodexHome -Force | Out-Null
  }

  $configTemp = Join-Path $CodexHome (".config.toml.tmp." + [Guid]::NewGuid().ToString("N"))
  $authTemp = Join-Path $CodexHome (".auth.json.tmp." + [Guid]::NewGuid().ToString("N"))
  try {
    $old = ""
    if (Test-Path -LiteralPath $ConfigFile -PathType Leaf) {
      $old = [System.IO.File]::ReadAllText($ConfigFile)
    }
    $content = Get-CodexConfig $old

    # 固定格式，Windows PowerShell 5.1 和 PowerShell 7 写出的内容一致。
    $escapedKey = ConvertTo-JsonText $ApiKey
    $json = "{`n  `"OPENAI_API_KEY`": `"$escapedKey`"`n}"
    Write-Utf8NoBom -Path $configTemp -Content $content
    Write-Utf8NoBom -Path $authTemp -Content ($json + "`n")

    Backup-IfExists $ConfigFile
    Backup-IfExists $AuthFile
    Move-Item -LiteralPath $configTemp -Destination $ConfigFile -Force
    Move-Item -LiteralPath $authTemp -Destination $AuthFile -Force
  } finally {
    Remove-Item -LiteralPath $configTemp -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath $authTemp -Force -ErrorAction SilentlyContinue
  }
}

Say "欢迎使用 MyAutoAPI 中转站 Codex 一键配置脚本。"
Say ""
Say "本脚本不会安装、修改任何软件，只会配置用户目录下 .codex 文件夹中的 config.toml 和 auth.json。"

Show-WritePlan
Say ""
Say "写入配置前会自动关闭 ChatGPT 软件和 Codex，正在运行的任务会中断。"
Say ""

$ApiKey = Get-CheckedApiKey

$StopResult = Stop-RunningClients
Write-CodexFiles -ApiKey $ApiKey

Say ""
Say "配置已经完成，可以关闭 PowerShell 了。"
Say ""
Report-RunningClients $StopResult
Say ""
Say "感谢您对 MyAutoAPI 中转站的支持，使用过程中如有问题请联系网站客服。"
