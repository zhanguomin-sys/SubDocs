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
  Say "检测到已有旧配置，将会在备份后再覆盖。"
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

function Get-WindowsTable {
  # 只读出原来 [windows] 段的 sandbox 值，这一段始终由脚本自己写，旧文件怎么写都不会写坏。
  $mode = ""
  if (Test-Path -LiteralPath $ConfigFile -PathType Leaf) {
    $inside = $false
    foreach ($line in ([System.IO.File]::ReadAllText($ConfigFile) -replace "`r`n", "`n").Split("`n")) {
      if ($line -match '^\s*\[([^\[\]]+)\]\s*(#.*)?$') {
        $inside = ($Matches[1].Trim() -ceq "windows")
        continue
      }
      if ($inside -and $line -match '^\s*sandbox\s*=\s*["''](elevated|unelevated)["'']\s*(#.*)?$') {
        $mode = $Matches[1]
      }
    }
  }
  if ($mode -eq "") {
    $mode = "unelevated"
    if (Test-Path -LiteralPath (Join-Path $CodexHome ".sandbox") -PathType Container) { $mode = "elevated" }
  }
  return "[windows]`nsandbox = `"$mode`""
}

function Write-CodexFiles([string]$ApiKey) {
  if (-not (Test-Path -LiteralPath $CodexHome)) {
    New-Item -ItemType Directory -Path $CodexHome -Force | Out-Null
  }

  $configTemp = Join-Path $CodexHome (".config.toml.tmp." + [Guid]::NewGuid().ToString("N"))
  $authTemp = Join-Path $CodexHome (".auth.json.tmp." + [Guid]::NewGuid().ToString("N"))
  try {
    $content = @"
model_provider = "OpenAI"
model = "$CodexModel"
review_model = "$CodexModel"
model_reasoning_effort = "high"
disable_response_storage = true
approval_policy = "never"
sandbox_mode = "danger-full-access"
network_access = "enabled"
windows_wsl_setup_acknowledged = true

[model_providers.OpenAI]
name = "OpenAI"
base_url = "$CodexBaseUrl"
wire_api = "responses"
requires_openai_auth = true

[features]
goals = true
image_generation = true
"@

    # Windows 上 Codex App 要先设置好沙箱才能选完全访问，设置结果记在 [windows] 段的 sandbox，
    # 所以沿用原来的值；没有时，App 设置过沙箱（有 .sandbox 文件夹）写 elevated，
    # 否则写不需要管理员权限的 unelevated。与快速接入助手写法一致。
    $content = $content + "`n`n" + (Get-WindowsTable)

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

# Codex App 底部的权限按钮读的是 .codex-global-state.json，不是 config.toml。
# 只改 App 选择“完全访问”时写入的三项，其它内容原样保留；文件不存在就新建，
# 文件无法识别就不动。必须在关闭 Codex 之后调用，否则 App 退出时会写回旧值。
# 不用 ConvertFrom-Json/ConvertTo-Json：5.1 版遇到只差大小写的键会报错，
# 写回时还会截断深层内容。7 版用 System.Text.Json，5.1 版用 JavaScriptSerializer。
function Grant-AppFullAccess {
  $stateFile = Join-Path $CodexHome ".codex-global-state.json"
  $temporary = Join-Path $CodexHome (".codex-global-state.json.tmp." + [Guid]::NewGuid().ToString("N"))
  $now = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
  try {
    $text = $null
    if (Test-Path -LiteralPath $stateFile -PathType Leaf) {
      $text = [System.IO.File]::ReadAllText($stateFile, [System.Text.Encoding]::UTF8)
    }
    $nodes = "System.Text.Json.Nodes.JsonNode" -as [type]
    if ($nodes) {
      if ($null -eq $text) { $text = "{}" }
      $state = [System.Text.Json.Nodes.JsonNode]::Parse($text)
      if ($state -isnot [System.Text.Json.Nodes.JsonObject]) { return }
      $atoms = $state["electron-persisted-atom-state"]
      if ($null -eq $atoms) {
        $atoms = [System.Text.Json.Nodes.JsonObject]::new()
        $state["electron-persisted-atom-state"] = $atoms
      } elseif ($atoms -isnot [System.Text.Json.Nodes.JsonObject]) {
        return
      }
      $hosts = $atoms["agent-mode-by-host-id"]
      if ($hosts -isnot [System.Text.Json.Nodes.JsonObject]) {
        $hosts = [System.Text.Json.Nodes.JsonObject]::new()
        $atoms["agent-mode-by-host-id"] = $hosts
      }
      $hosts["local"] = [System.Text.Json.Nodes.JsonValue]::Create([string]"full-access")
      $atoms["permission-selection-by-host-id:local"] = [System.Text.Json.Nodes.JsonNode]::Parse('{"kind":"agent-mode","agentMode":"full-access"}')
      $atoms["full-access-warning-dismissed-at-v2"] = [System.Text.Json.Nodes.JsonValue]::Create([long]$now)
      $options = [System.Text.Json.JsonSerializerOptions]::new()
      $options.Encoder = [System.Text.Encodings.Web.JavaScriptEncoder]::UnsafeRelaxedJsonEscaping
      $output = $state.ToJsonString($options)
    } else {
      Add-Type -AssemblyName System.Web.Extensions
      $serializer = New-Object System.Web.Script.Serialization.JavaScriptSerializer
      $serializer.MaxJsonLength = [int]::MaxValue
      $serializer.RecursionLimit = 1000
      if ($null -eq $text) {
        $state = New-Object 'System.Collections.Generic.Dictionary[string,object]'
      } else {
        $state = $serializer.DeserializeObject($text)
      }
      if ($state -isnot [System.Collections.Generic.Dictionary[string, object]]) { return }
      if (-not $state.ContainsKey("electron-persisted-atom-state")) {
        $state["electron-persisted-atom-state"] = New-Object 'System.Collections.Generic.Dictionary[string,object]'
      }
      $atoms = $state["electron-persisted-atom-state"]
      if ($atoms -isnot [System.Collections.Generic.Dictionary[string, object]]) { return }
      $hosts = $null
      if ($atoms.ContainsKey("agent-mode-by-host-id")) { $hosts = $atoms["agent-mode-by-host-id"] }
      if ($hosts -isnot [System.Collections.Generic.Dictionary[string, object]]) {
        $hosts = New-Object 'System.Collections.Generic.Dictionary[string,object]'
        $atoms["agent-mode-by-host-id"] = $hosts
      }
      $hosts["local"] = "full-access"
      $selection = New-Object 'System.Collections.Generic.Dictionary[string,object]'
      $selection["kind"] = "agent-mode"
      $selection["agentMode"] = "full-access"
      $atoms["permission-selection-by-host-id:local"] = $selection
      $atoms["full-access-warning-dismissed-at-v2"] = [long]$now
      $output = $serializer.Serialize($state)
    }
    Write-Utf8NoBom -Path $temporary -Content $output
    Move-Item -LiteralPath $temporary -Destination $stateFile -Force
  } catch {
    # config.toml 已经是完全访问，这一步失败不影响使用。
  } finally {
    Remove-Item -LiteralPath $temporary -Force -ErrorAction SilentlyContinue
  }
}

Say "欢迎使用 MyAutoAPI 中转站 Codex 一键配置脚本。"
Say ""
Say "本脚本不会安装、修改任何软件，只会配置用户目录下 .codex 文件夹中的 config.toml、auth.json 和 Codex App 权限设置。"

Show-WritePlan
Say ""
Say "写入配置前会自动关闭 ChatGPT 软件和 Codex，正在运行的任务会中断。"
Say ""

$ApiKey = Get-CheckedApiKey

$StopResult = Stop-RunningClients
Write-CodexFiles -ApiKey $ApiKey
Grant-AppFullAccess

Say ""
Say "配置已经完成，可以关闭 PowerShell 了。"
Say ""
Report-RunningClients $StopResult
Say ""
Say "感谢您对 MyAutoAPI 中转站的支持，使用过程中如有问题请联系网站客服。"
