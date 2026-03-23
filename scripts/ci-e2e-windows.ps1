# CI / native Windows: run clipboard WebP + file-drop E2E (same cases as macOS / WSL2).
# Entry point: powershell.exe -Sta -NoProfile -ExecutionPolicy Bypass -File scripts/ci-e2e-windows.ps1
#Requires -Version 5.1
$ErrorActionPreference = 'Stop'

& (Join-Path $PSScriptRoot 'ci-e2e-windows-clipboard-webp.ps1')
& (Join-Path $PSScriptRoot 'ci-e2e-windows-clipboard-filedrop.ps1')
Write-Host 'ci-e2e-windows: all cases passed'
