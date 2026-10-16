param([string]$GeneratedScript)
$ErrorActionPreference='Stop'
$script:assertions=0
function Assert($Value) { if (-not $Value) { throw "Assertion $($script:assertions+1) failed" }; $script:assertions++ }
$text=[IO.File]::ReadAllText($GeneratedScript)
$start=$text.IndexOf('function Get-RunningCodexClients {')
$end=$text.IndexOf('function Get-TomlString(')
# The generated block begins with the exact original process functions and New-OrdinalMap.
Invoke-Expression $text.Substring($start,$end-$start)
$script:own=[int]$PID
$script:stopped=New-Object 'System.Collections.Generic.List[int]'
$script:messages=New-Object 'System.Collections.Generic.List[string]'
function Say([string]$Message) { $script:messages.Add($Message) }
function Get-Process {
  param($Id,$Name,$ErrorAction)
  if ($null -ne $Id) { return [pscustomobject]@{SessionId=7} }
  return @(
    [pscustomobject]@{Id=$script:own; ProcessName='codex'; SessionId=7; HasExited=$true},
    [pscustomobject]@{Id=7000; ProcessName='codex'; SessionId=7; HasExited=$true},
    [pscustomobject]@{Id=7001; ProcessName='chatgpt'; SessionId=7; HasExited=$true},
    [pscustomobject]@{Id=7002; ProcessName='codex'; SessionId=8; HasExited=$false}
  )
}
function Get-CimInstance {
  param($ClassName,$Filter,$ErrorAction)
  $parent=if ($Filter -eq "ProcessId = $script:own") { 7000 } else { 0 }
  return [pscustomobject]@{ParentProcessId=$parent}
}
function Stop-Process {
  param([Parameter(ValueFromPipeline=$true)]$InputObject,[switch]$Force)
  process { $script:stopped.Add($InputObject.Id) }
}
function Start-Sleep { param($Milliseconds) }
$result=Stop-RunningClients
Assert $result.Codex; Assert $result.ChatGPT; Assert $result.Left
Assert ($script:stopped.Count -eq 1 -and $script:stopped[0] -eq 7001)
Report-RunningClients $result
Assert ($script:messages[-1] -like '未能自动关闭*')
function Get-RunningCodexClients { return @([pscustomobject]@{Id=7001; ProcessName='codex'; HasExited=$true}) }
$result=Stop-RunningClients
Assert $result.Codex; Assert (-not $result.Left)
Report-RunningClients $result
Assert ($script:messages[-1] -like '已自动关闭*')
function Get-RunningCodexClients { return @() }
$result=Stop-RunningClients
Assert (-not $result.Codex -and -not $result.ChatGPT -and -not $result.Left)
Report-RunningClients $result
Assert ($script:messages[-1] -like '当前未检测到*')
function Get-RunningCodexClients { return @([pscustomobject]@{Id=7001; ProcessName='codex'; HasExited=$false}) }
$result=Stop-RunningClients
Assert $result.Left
function Get-RunningCodexClients { throw 'simulated access failure' }
$result=Stop-RunningClients
Assert $result.Left
Write-Output "PowerShell processes: $script:assertions assertions passed; no real process was stopped"
