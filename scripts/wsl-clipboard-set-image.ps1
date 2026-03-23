# Set the Windows clipboard image from a PNG (or other format System.Drawing supports).
# Run with Windows PowerShell using -Sta (required for System.Windows.Forms.Clipboard).
param(
    [Parameter(Mandatory = $true)]
    [string]$ImagePath
)
Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
$img = [System.Drawing.Image]::FromFile($ImagePath)
try {
    [System.Windows.Forms.Clipboard]::SetImage($img)
} finally {
    $img.Dispose()
}
