# Paste cURL Download

本機小工具：貼上 Chrome **Copy as cURL (bash)**，自動把 video 嘅 `Range` 改成 `bytes=0-`（全片，唔係 4 MB chunk），再用 `curl` 下載到 Downloads。

**唔會執行你貼上嘅 shell 字串**（避免 command injection）；只抽取 `https` URL、`-H` headers、`-b` cookies。

## Windows（你而家呢部機）

1. 雙擊 `Start.bat`
2. 喺 Chrome Network → 對 MP4 右鍵 → **Copy** → **Copy as cURL (bash)**
3. 貼入視窗（`Ctrl+A` 全選、`Ctrl+V` 貼上）
4. 撳 **Download**

有 Python 3 會開 tk 視窗；冇就自動用 PowerShell 視窗。兩邊行為一樣。

需要 `curl.exe`（Windows 10+ 或 Git for Windows）。

## Git Bash / CLI

```bash
cd tools/paste-curl-download
python3 paste_curl_download.py --cli
# paste, then Ctrl+D
```

或：

```bash
python3 paste_curl_download.py --cli --paste-file curl.txt -o ~/Downloads
```

## 注意

- Zoom signed URL 會過期；cookie／`cf_clearance` 綁瀏覽器同 IP，唔好換 VPN。
- 貼上內容含 session cookie，唔好分享 `curl.conf` 或截圖。
- 長片檔通常遠大於 5 MB；細過呢個數即 Range／URL 仍有問題。
