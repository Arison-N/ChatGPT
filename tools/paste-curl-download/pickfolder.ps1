param(
    [Parameter(Mandatory = $true)][string]$OutPath,
    [string]$Initial = "",
    [Parameter(Mandatory = $true)][string]$CsPath,
    [string]$Title = "File saving location"
)
$ErrorActionPreference = "Stop"
if (-not $Title) { $Title = "File saving location" }
$path = $null
try {
    Add-Type -LiteralPath $CsPath -ErrorAction Stop
    $path = [ExplorerFolderPicker]::Pick($Title, $Initial)
} catch {
    Add-Type -AssemblyName System.Windows.Forms
    $d = New-Object System.Windows.Forms.FolderBrowserDialog
    $d.Description = $Title
    $d.ShowNewFolderButton = $true
    if ($Initial) { $d.SelectedPath = $Initial }
    if ($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) {
        $path = $d.SelectedPath
    }
}
if ($path) {
    $utf8 = New-Object System.Text.UTF8Encoding $false
    [System.IO.File]::WriteAllText($OutPath, $path, $utf8)
}
