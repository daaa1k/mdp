# CI / native Windows: file-drop clipboard -> mdp local -> byte-identical outputs.
# Uses the same fixtures and cases as macOS / WSL2 E2E. Run with Windows PowerShell (-Sta).
#Requires -Version 5.1
$ErrorActionPreference = 'Stop'

# Avoid Get-FileHash: not available in some Windows PowerShell -Sta / CI environments.
function Get-FileSha256Hex {
    param([Parameter(Mandatory)][string]$LiteralPath)
    $sha = [System.Security.Cryptography.SHA256]::Create()
    try {
        $fs = [System.IO.File]::OpenRead($LiteralPath)
        try {
            $bytes = $sha.ComputeHash($fs)
        } finally {
            $fs.Dispose()
        }
    } finally {
        $sha.Dispose()
    }
    return ([System.BitConverter]::ToString($bytes)) -replace '-', ''
}

function Assert-OutCount {
    param([Parameter(Mandatory)][string]$OutDir, [Parameter(Mandatory)][int]$Want)
    $items = @(Get-ChildItem -LiteralPath $OutDir -File -ErrorAction SilentlyContinue)
    if ($items.Count -ne $Want) {
        Write-Error "expected $Want file(s) in $OutDir, got $($items.Count)"
    }
}

function Assert-OutputsMatchSources {
    param([Parameter(Mandatory)][string]$OutDir, [Parameter(Mandatory)][string[]]$Sources)
    $outs = @(Get-ChildItem -LiteralPath $OutDir -File -ErrorAction SilentlyContinue)
    if ($outs.Count -ne $Sources.Count) {
        Write-Error "output count $($outs.Count) != source count $($Sources.Count)"
    }
    foreach ($src in $Sources) {
        $hashSrc = Get-FileSha256Hex -LiteralPath $src
        $found = $false
        foreach ($o in $outs) {
            if ((Get-FileSha256Hex -LiteralPath $o.FullName) -eq $hashSrc) {
                $found = $true
                break
            }
        }
        if (-not $found) {
            Write-Error "no output matches source $(Split-Path -Leaf $src)"
        }
    }
}

function Set-ClipboardFileDropFromPaths {
    param([Parameter(Mandatory)][string[]]$WinPaths)
    $listPath = Join-Path $env:TEMP ('mdp-filedrop-' + [Guid]::NewGuid().ToString('N') + '.txt')
    $utf8 = New-Object System.Text.UTF8Encoding $false
    [System.IO.File]::WriteAllLines($listPath, $WinPaths, $utf8)
    try {
        $SetDropPs1 = Join-Path $PSScriptRoot 'wsl-clipboard-set-filedrop.ps1'
        & powershell.exe -Sta -NoProfile -ExecutionPolicy Bypass -File $SetDropPs1 -PathsFile $listPath
    } finally {
        Remove-Item -LiteralPath $listPath -Force -ErrorAction SilentlyContinue
    }
}

$RepoRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$FixtureDir = Join-Path $RepoRoot 'e2e\fixtures\filedrop'
foreach ($name in @('one.png', 'one.webp', 'two.png', 'two.webp')) {
    $p = Join-Path $FixtureDir $name
    if (-not (Test-Path -LiteralPath $p)) {
        Write-Error "missing fixture $p"
    }
}

$TmpRoot = Join-Path ([System.IO.Path]::GetTempPath()) ('mdp-ci-e2e-filedrop-' + [Guid]::NewGuid().ToString('N'))
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

    $MdpExe = Join-Path $TmpRoot 'mdp.exe'
    Push-Location $RepoRoot
    try {
        & go build -o $MdpExe .
        if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
    } finally {
        Pop-Location
    }

    function Invoke-Mdp {
        if (Test-Path -LiteralPath $OutDir) {
            Remove-Item -LiteralPath $OutDir -Recurse -Force
        }
        New-Item -ItemType Directory -Path $OutDir -Force | Out-Null
        Push-Location $TmpRoot
        try {
            & $MdpExe --backend local
            if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
        } finally {
            Pop-Location
        }
    }

    # --- Single PNG ---
    $case1 = Join-Path $TmpRoot 'case1.png'
    Copy-Item -LiteralPath (Join-Path $FixtureDir 'one.png') -Destination $case1 -Force
    Set-ClipboardFileDropFromPaths -WinPaths @($case1)
    Invoke-Mdp
    Assert-OutCount -OutDir $OutDir -Want 1
    Assert-OutputsMatchSources -OutDir $OutDir -Sources @($case1)
    Write-Host 'e2e ok: single PNG'

    # --- Single WebP ---
    $case2 = Join-Path $TmpRoot 'case2.webp'
    Copy-Item -LiteralPath (Join-Path $FixtureDir 'one.webp') -Destination $case2 -Force
    Set-ClipboardFileDropFromPaths -WinPaths @($case2)
    Invoke-Mdp
    Assert-OutCount -OutDir $OutDir -Want 1
    Assert-OutputsMatchSources -OutDir $OutDir -Sources @($case2)
    Write-Host 'e2e ok: single WebP'

    # --- Multiple (PNG + WebP) ---
    $ma = Join-Path $TmpRoot 'm_a.png'
    $mb = Join-Path $TmpRoot 'm_b.webp'
    Copy-Item -LiteralPath (Join-Path $FixtureDir 'two.png') -Destination $ma -Force
    Copy-Item -LiteralPath (Join-Path $FixtureDir 'two.webp') -Destination $mb -Force
    Set-ClipboardFileDropFromPaths -WinPaths @($ma, $mb)
    Invoke-Mdp
    Assert-OutCount -OutDir $OutDir -Want 2
    Assert-OutputsMatchSources -OutDir $OutDir -Sources @($ma, $mb)
    Write-Host 'e2e ok: multiple PNG+WebP'

    Write-Host 'ci-e2e-windows-clipboard-filedrop: all cases passed'
} finally {
    Remove-Item -LiteralPath $TmpRoot -Recurse -Force -ErrorAction SilentlyContinue
}
