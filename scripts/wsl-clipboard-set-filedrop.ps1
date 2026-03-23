# Set the Windows clipboard file-drop list from a UTF-8 text file (one Windows path per line).
# Run with Windows PowerShell using -Sta (required for System.Windows.Forms.Clipboard).
param(
    [Parameter(Mandatory = $true)]
    [string]$PathsFile
)
Add-Type -AssemblyName System.Windows.Forms
$lines = Get-Content -LiteralPath $PathsFile -Encoding UTF8
$col = New-Object System.Collections.Specialized.StringCollection
foreach ($line in $lines) {
    $t = $line.Trim()
    if ($t -ne '') {
        [void]$col.Add($t)
    }
}
[System.Windows.Forms.Clipboard]::SetFileDropList($col)
