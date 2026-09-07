# Paste Chrome "Copy as cURL (bash)" and download the full file.
# Double-click Start.bat, or: powershell -ExecutionPolicy Bypass -File PasteCurlDownload.ps1
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

Add-Type -AssemblyName System.Windows.Forms
Add-Type -AssemblyName System.Drawing
[System.Windows.Forms.Application]::EnableVisualStyles()

function Get-DownloadDir {
    $dir = Join-Path $env:USERPROFILE "Downloads"
    if (Test-Path -LiteralPath $dir) { return $dir }
    return $env:USERPROFILE
}

function ConvertTo-NormalizedPaste([string]$Text) {
    $t = $Text -replace "`r`n", "`n" -replace "`r", "`n"
    $t = $t.Trim()
    $t = [regex]::Replace($t, '(?m)^\$\s*', "")
    $t = [regex]::Replace($t, '^(?:cd\s+\S+\s*&&\s*)+', "")
    $t = [regex]::Replace($t, '^(?:~/[^\s]*|/[^\s]+|[A-Za-z]:\\[^\s]+)\s*&&\s*', "")
    $t = [regex]::Replace($t, "\\\s*\n", " ")
    $t = [regex]::Replace($t, "\^\s*\n", " ")
    return $t.Trim()
}

function ConvertTo-LectureFilename([string]$UserName, [string]$Detected) {
    $name = $UserName.Trim()
    if ([string]::IsNullOrWhiteSpace($name)) { $name = $Detected.Trim() }
    if ([string]::IsNullOrWhiteSpace($name)) { throw "請輸入課堂檔名" }
    $name = $name.Replace("\", "/")
    $idx = $name.LastIndexOf("/")
    if ($idx -ge 0) { $name = $name.Substring($idx + 1) }
    $name = [regex]::Replace($name.Trim(), '[<>:"/\\|?*]', "_")
    if ([string]::IsNullOrWhiteSpace($name) -or $name -eq "." -or $name -eq "..") {
        throw "課堂檔名無效"
    }
    if (-not [System.IO.Path]::HasExtension($name)) { $name += ".mp4" }
    return $name
}

function Get-FilenameFromUrl([string]$Url) {
    $noQuery = $Url.Split("?")[0]
    $name = [System.Uri]::UnescapeDataString(($noQuery.Split("/") | Select-Object -Last 1))
    if ([string]::IsNullOrWhiteSpace($name)) { $name = "download.bin" }
    $name = [regex]::Replace($name, '[<>:"/\\|?*]', "_")
    return $name
}
    $noQuery = $Url.Split("?")[0]
    $name = [System.Uri]::UnescapeDataString(($noQuery.Split("/") | Select-Object -Last 1))
    if ([string]::IsNullOrWhiteSpace($name)) { $name = "download.bin" }
    $name = [regex]::Replace($name, '[<>:"/\\|?*]', "_")
    return $name
}

function ConvertFrom-CurlPaste([string]$Text) {
    $blob = ConvertTo-NormalizedPaste $Text
    if ([string]::IsNullOrWhiteSpace($blob)) {
        throw "Empty paste."
    }

    $url = $null
    $urlFlag = [regex]::Match($blob, "--url\s+(['""])(?<url>https://.+?)\1", "IgnoreCase")
    if ($urlFlag.Success) {
        $url = $urlFlag.Groups["url"].Value
    }
    else {
        $curlUrl = [regex]::Match($blob, "\bcurl(?:\.exe)?\s+(?:-[A-Za-z]\s+)*['""](?<url>https://[^'""]+)['""]", "IgnoreCase")
        if ($curlUrl.Success) {
            $url = $curlUrl.Groups["url"].Value
        }
        else {
            $https = [regex]::Matches($blob, "https://[^\s'""\\]+")
            if ($https.Count -gt 0) {
                $url = ($https | Sort-Object Length -Descending | Select-Object -First 1).Value
            }
        }
    }

    $url = [string]$url
    $url = $url.Trim().TrimEnd("\")
    if (-not $url.StartsWith("https://")) {
        throw "No https URL found. Paste Chrome Copy as cURL (bash)."
    }

    $headers = New-Object System.Collections.Generic.List[string]
    foreach ($m in [regex]::Matches($blob, "(?:-H|--header)\s+(['""])(?<val>.*?)\1", "IgnoreCase")) {
        [void]$headers.Add($m.Groups["val"].Value.Trim())
    }

    $cookie = ""
    $cookieMatches = [regex]::Matches($blob, "(?:-b|--cookie)\s+(['""])(?<val>.*?)\1", "IgnoreCase")
    if ($cookieMatches.Count -gt 0) {
        $cookie = $cookieMatches[$cookieMatches.Count - 1].Groups["val"].Value.Trim()
    }

    $rewritten = New-Object System.Collections.Generic.List[string]
    $seen = New-Object "System.Collections.Generic.HashSet[string]"
    $hasRange = $false
    foreach ($h in $headers) {
        $idx = $h.IndexOf(":")
        if ($idx -lt 0) { continue }
        $name = $h.Substring(0, $idx)
        $value = $h.Substring($idx + 1)
        $lname = $name.Trim().ToLowerInvariant()
        if (-not $seen.Add($lname)) { continue }
        if ($lname -eq "range") {
            [void]$rewritten.Add("Range: bytes=0-")
            $hasRange = $true
        }
        elseif ($lname -eq "cookie" -and -not $cookie) {
            $cookie = $value.Trim()
            [void]$rewritten.Add($h)
        }
        else {
            [void]$rewritten.Add($h)
        }
    }
    if (-not $hasRange) {
        [void]$rewritten.Add("Range: bytes=0-")
    }

    return [pscustomobject]@{
        Url      = $url
        Headers  = $rewritten
        Cookie   = $cookie
        Filename = Get-FilenameFromUrl $url
    }
}

function Escape-CurlConfig([string]$Value) {
    return $Value.Replace('\', '\\').Replace('"', '\"')
}

function Get-CurlExe {
    $git = "C:\Program Files\Git\mingw64\bin\curl.exe"
    if (Test-Path -LiteralPath $git) { return $git }
    $cmd = Get-Command curl.exe -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    throw "curl.exe not found. Install Git for Windows."
}

function Start-CurlDownload($Parsed, [string]$DestPath) {
    $curl = Get-CurlExe
    $tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("paste-curl-" + [guid]::NewGuid().ToString("n"))
    New-Item -ItemType Directory -Path $tmp | Out-Null
    $conf = Join-Path $tmp "curl.conf"
    $destEsc = Escape-CurlConfig $DestPath
    $urlEsc = Escape-CurlConfig $Parsed.Url
    $lines = @(
        "location",
        "fail",
        'retry = "3"',
        'continue-at = "-"',
        "output = `"$destEsc`"",
        "url = `"$urlEsc`""
    )
    if ($Parsed.Cookie) {
        $c = Escape-CurlConfig $Parsed.Cookie
        $lines += "cookie = `"$c`""
    }
    foreach ($h in $Parsed.Headers) {
        $hh = Escape-CurlConfig $h
        $lines += "header = `"$hh`""
    }
    $utf8 = New-Object System.Text.UTF8Encoding $false
    [System.IO.File]::WriteAllText($conf, ($lines -join "`n") + "`n", $utf8)

    $p = Start-Process -FilePath $curl -ArgumentList @("-K", $conf) -PassThru -NoNewWindow
    return [pscustomobject]@{ Process = $p; TempDir = $tmp }
}

$form = New-Object System.Windows.Forms.Form
$form.Text = "Zoom-loader"
$form.Size = New-Object System.Drawing.Size(920, 680)
$form.StartPosition = "CenterScreen"
$form.MinimumSize = New-Object System.Drawing.Size(720, 520)

$hint = New-Object System.Windows.Forms.Label
$hint.Text = "Enter custom file name and paste the Chrome Copy as cURL (bash)."
$hint.AutoSize = $false
$hint.Height = 36
$hint.Dock = "Top"
$hint.Padding = New-Object System.Windows.Forms.Padding(12, 8, 12, 0)
$form.Controls.Add($hint)

$box = New-Object System.Windows.Forms.TextBox
$box.Multiline = $true
$box.ScrollBars = "Both"
$box.AcceptsReturn = $true
$box.Font = New-Object System.Drawing.Font("Consolas", 9)
$box.MaxLength = 0
$box.Dock = "Fill"
$box.WordWrap = $false
$box.Add_KeyDown({
        if ($_.Control -and $_.KeyCode -eq "A") {
            $box.SelectAll()
            $_.SuppressKeyPress = $true
        }
    })

$bottom = New-Object System.Windows.Forms.Panel
$bottom.Dock = "Bottom"
$bottom.Height = 210

$dirLabel = New-Object System.Windows.Forms.Label
$dirLabel.Text = "File saving location"
$dirLabel.Location = New-Object System.Drawing.Point(12, 12)
$dirLabel.AutoSize = $true
$bottom.Controls.Add($dirLabel)

$dirBox = New-Object System.Windows.Forms.TextBox
$dirBox.ReadOnly = $true
$dirBox.TabStop = $false
$dirBox.Text = Get-DownloadDir
$dirBox.Location = New-Object System.Drawing.Point(80, 8)
$dirBox.Width = 680
$bottom.Controls.Add($dirBox)

$browse = New-Object System.Windows.Forms.Button
$browse.Text = "Choose folder"
$browse.Location = New-Object System.Drawing.Point(770, 6)
$browse.Add_Click({
        $dlg = New-Object System.Windows.Forms.FolderBrowserDialog
        $dlg.SelectedPath = $dirBox.Text
        if ($dlg.ShowDialog() -eq "OK") { $dirBox.Text = $dlg.SelectedPath }
    })
$bottom.Controls.Add($browse)

$nameLabel = New-Object System.Windows.Forms.Label
$nameLabel.Text = "File name:"
$nameLabel.Location = New-Object System.Drawing.Point(12, 44)
$nameLabel.AutoSize = $true
$bottom.Controls.Add($nameLabel)

$nameBox = New-Object System.Windows.Forms.TextBox
$nameBox.Location = New-Object System.Drawing.Point(80, 40)
$nameBox.Width = 680
$bottom.Controls.Add($nameBox)

$dl = New-Object System.Windows.Forms.Button
$dl.Text = "Download"
$dl.Location = New-Object System.Drawing.Point(80, 74)
$clear = New-Object System.Windows.Forms.Button
$clear.Text = "Clear"
$clear.Location = New-Object System.Drawing.Point(180, 74)

$log = New-Object System.Windows.Forms.TextBox
$log.Multiline = $true
$log.ReadOnly = $true
$log.ScrollBars = "Vertical"
$log.Location = New-Object System.Drawing.Point(12, 112)
$log.Width = 870
$log.Height = 88
$bottom.Controls.Add($log)

function Write-Log([string]$Msg) {
    $log.AppendText($Msg + [Environment]::NewLine)
}

$box.Add_TextChanged({
        if ($box.Text -notmatch "https://") { return }
        try {
            $parsed = ConvertFrom-CurlPaste $box.Text
            if ([string]::IsNullOrWhiteSpace($nameBox.Text)) {
                $nameBox.Text = $parsed.Filename
            }
        }
        catch { }
    })

$script:active = $null
$timer = New-Object System.Windows.Forms.Timer
$timer.Interval = 500
$timer.Add_Tick({
        if (-not $script:active) { return }
        $proc = $script:active.Process
        $dest = $script:active.Dest
        $tmp = $script:active.TempDir
        if (-not $proc.HasExited) { return }
        $timer.Stop()
        $code = $proc.ExitCode
        Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
        $script:active = $null
        $dl.Enabled = $true
        if ($code -ne 0) {
            Write-Log "curl exited $code"
            [System.Windows.Forms.MessageBox]::Show("curl exited $code", "Error") | Out-Null
            return
        }
        if (-not (Test-Path -LiteralPath $dest)) {
            Write-Log "File not found after download"
            [System.Windows.Forms.MessageBox]::Show("File not found:`n$dest", "Error") | Out-Null
            return
        }
        $size = (Get-Item -LiteralPath $dest).Length
        Write-Log "Saved $size bytes"
        if ($size -lt 5MB) {
            Write-Log "Warning: file < 5 MB. Long Zoom recordings are usually much larger."
        }
        [System.Windows.Forms.MessageBox]::Show("Saved:`n$dest`n$size bytes", "Done") | Out-Null
    })

$dl.Add_Click({
        try {
            $parsed = ConvertFrom-CurlPaste $box.Text
            $filename = ConvertTo-LectureFilename $nameBox.Text $parsed.Filename
            $destDir = $dirBox.Text.Trim()
            if (-not (Test-Path -LiteralPath $destDir)) {
                New-Item -ItemType Directory -Path $destDir | Out-Null
            }
            $dest = Join-Path $destDir $filename
            Write-Log "Downloading → $dest"
            Write-Log ($parsed.Url.Split("?")[0])
            $dl.Enabled = $false
            $job = Start-CurlDownload $parsed $dest
            $script:active = [pscustomobject]@{
                Process = $job.Process
                TempDir = $job.TempDir
                Dest    = $dest
            }
            $timer.Start()
        }
        catch {
            $dl.Enabled = $true
            Write-Log $_.Exception.Message
            [System.Windows.Forms.MessageBox]::Show($_.Exception.Message, "Error") | Out-Null
        }
    })

$clear.Add_Click({ $box.Clear(); $nameBox.Clear(); $log.Clear() })
$bottom.Controls.Add($dl)
$bottom.Controls.Add($clear)

$form.Add_FormClosing({
        $timer.Stop()
        if ($script:active -and -not $script:active.Process.HasExited) {
            try { $script:active.Process.Kill() } catch { }
        }
    })

$form.Controls.Add($bottom)
$form.Controls.Add($box)
[void]$form.ShowDialog()
