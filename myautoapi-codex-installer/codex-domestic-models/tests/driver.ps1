param([string]$InputPath)
$ErrorActionPreference='Stop'
. (Join-Path $PSScriptRoot '../src/core.ps1')
. (Join-Path $PSScriptRoot '../src/windows-host.ps1')
$cases=ConvertFrom-Json -InputObject ([IO.File]::ReadAllText($InputPath))
$results=New-Object 'System.Collections.Generic.List[object]'
foreach ($test in $cases) {
  try {
    $info=Get-ConfigInfo $test.config
    $merged=Merge-ModelCatalog $test.existing $test.app $test.domestic $test.managed $info
    $config=Set-ModelConfig $test.config $info $test.path $merged.Efforts
    $sourcePath=if ($info.Values.ContainsKey('model_catalog_json')) { Get-TomlString $info.Values['model_catalog_json'] } else { $null }
    $results.Add(@{ok=$true; catalog=$merged.Catalog; managed=$merged.Managed; efforts=$merged.Efforts; sourcePath=$sourcePath; config=$config})
  } catch {
    # Fixture data only: expose error for developer diagnosis without printing real user config.
    [Console]::Error.WriteLine($_.ScriptStackTrace+': '+$_.Exception.Message)
    $results.Add(@{ok=$false})
  }
}
ConvertTo-Json -InputObject @($results.ToArray()) -Depth 100 -Compress
