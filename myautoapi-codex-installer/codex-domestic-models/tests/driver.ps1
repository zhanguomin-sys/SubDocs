# Runs the Windows importer (or uninstaller) on every case directory under CasesRoot.
param([string]$CasesRoot, [string]$Action='import')
$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot '../src/core.ps1')
. (Join-Path $PSScriptRoot '../src/windows-host.ps1')
$DomesticCatalog=ConvertFrom-Json -InputObject ([IO.File]::ReadAllText((Join-Path $CasesRoot 'domestic.json')))
$BundledOfficialCatalog=ConvertFrom-Json -InputObject ([IO.File]::ReadAllText((Join-Path $PSScriptRoot '../src/gpt-catalog.json')))
$BundledOfficialVersion='0.162.1'
$results=[ordered]@{}
foreach ($dir in (Get-ChildItem -LiteralPath $CasesRoot -Directory)) {
  try {
    if ($Action -ceq 'uninstall') { $r=Remove-ImportedModels $dir.FullName $DomesticCatalog; $results[$dir.Name]=@{ok=$true; changed=$r.Changed; count=$r.Count} }
    else { $r=Invoke-ModelImport $dir.FullName $DomesticCatalog; $results[$dir.Name]=@{ok=$true; usedCache=$r.UsedCache; count=$r.Count} }
  } catch {
    $results[$dir.Name]=@{ok=$false; error=(Format-ModelImportError $_)}
  }
}
ConvertTo-Json -InputObject $results -Depth 10 -Compress
