# CI / native Windows: clipboard image fixture -> mdp local -> assert WebP output.
# Uses the same fixtures as macOS / WSL2 E2E. Run with Windows PowerShell (-Sta).
#Requires -Version 5.1
$ErrorActionPreference = 'Stop'

function Test-WebPFile {
    param([Parameter(Mandatory)][string]$LiteralPath)
    $fs = [System.IO.File]::OpenRead($LiteralPath)
    try {
        $buf = New-Object byte[] 12
        $n = $fs.Read($buf, 0, 12)
        if ($n -lt 12) { return $false }
        $riff = [System.Text.Encoding]::ASCII.GetString($buf[0..3])
        $webp = [System.Text.Encoding]::ASCII.GetString($buf[8..11])
        return ($riff -eq 'RIFF' -and $webp -eq 'WEBP')
    } finally {
        $fs.Dispose()
    }
}

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$FixtureDir = Join-Path $RepoRoot 'e2e\fixtures\clipboard-snapshot'
$KindPath = Join-Path $FixtureDir 'KIND'
$PayloadPath = Join-Path $FixtureDir 'payload'
if (-not (Test-Path -LiteralPath $KindPath) -or -not (Test-Path -LiteralPath $PayloadPath)) {
    Write-Error "missing fixture under $FixtureDir"
}

$TmpRoot = Join-Path ([System.IO.Path]::GetTempPath()) ('mdp-ci-e2e-webp-' + [Guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $TmpRoot -Force | Out-Null
try {
    $OutDir = Join-Path $TmpRoot 'out'
    New-Item -ItemType Directory -Path $OutDir -Force | Out-Null

    $MdpYaml = Join-Path $TmpRoot '.mdp.yaml'
    @'
backend: local
local:
  dir: ./out
'@ | Set-Content -LiteralPath $MdpYaml -Encoding UTF8

    $ClipPng = Join-Path $TmpRoot 'clip.png'
    Copy-Item -LiteralPath $PayloadPath -Destination $ClipPng -Force

    $SetImagePs1 = Join-Path $PSScriptRoot 'wsl-clipboard-set-image.ps1'
    & powershell.exe -Sta -NoProfile -ExecutionPolicy Bypass -File $SetImagePs1 -ImagePath $ClipPng

    $MdpExe = Join-Path $TmpRoot 'mdp.exe'
    Push-Location $RepoRoot
    try {
        & go build -o $MdpExe .
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    } finally {
        Pop-Location
    }

    Push-Location $TmpRoot
    try {
        & $MdpExe --backend local
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    } finally {
        Pop-Location
    }

    $outs = @(Get-ChildItem -LiteralPath $OutDir -File -ErrorAction SilentlyContinue)
    if ($outs.Count -eq 0) {
        Write-Error "no files written under $OutDir"
    }
    $OutFile = ($outs | Sort-Object LastWriteTime -Descending | Select-Object -First 1).FullName

    if (-not (Test-WebPFile -LiteralPath $OutFile)) {
        Write-Error "expected WebP output, got: $OutFile"
    }
    Write-Host "e2e ok: $OutFile (WebP)"
} finally {
    Remove-Item -LiteralPath $TmpRoot -Recurse -Force -ErrorAction SilentlyContinue
}
