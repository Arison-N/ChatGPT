# Zoom-loader

Paste Chrome **Copy as cURL (bash)** and save a Zoom recording under your own file name.

## Windows: double-click `.exe`

`tools/paste-curl-download/dist/Zoom-loader.exe` (or `Start.bat`)

1. **File name:** e.g. `yyyy-mm-dd_name_index` (`.mp4` is added if missing)
2. **File saving location:** Choose folder
3. Paste Copy as cURL (bash)
4. Download

Windows 用 **WebView2** 喺 `Zoom-loader.exe` 入面開窗（Taskbar / Task Manager 顯示 Zoom-loader，唔係 Google Chrome）。未裝 WebView2 Runtime 先會 fallback 去 Chrome `--app`。

UI 跟 Windows 顯示語言自動切，右上可選 English (ENG) / Traditional Chinese (ZH-hk) / Simplified Chinese (ZH-cn)。字體用 Google Noto Sans / Noto Sans HK / Noto Sans SC。Choose folder 用 Windows Explorer dialog，唔再開 PowerShell 視窗。下載完成後「開啟檔案位置」用 Shell API 將 Explorer 帶到最前並選中檔案（含中文路徑）。

**Check for updates** / **Update** pull a new `Zoom-loader.exe` from GitHub and restart. First install of this build is still a manual download; later changes can be applied in-app.

Rebuild:

```bash
cd tools/paste-curl-download
bash build-windows.sh
```


## 課堂檔名規則

- 你輸入嘅名稱就係儲存檔名
- 空白則用 curl 入面偵測到嘅原檔名
- 無副檔名 → `.mp4`
- 唔合法字元 `<>:"/\|?*` 會被取代
- 忽略路徑，只取檔名（唔會寫出 Downloads 以外）

## 其他啟動方式

Python GUI／CLI 同 PowerShell 後備仍可用（見 `Start.bat` fallback）。需要 `curl.exe` 嘅只有 fallback；`.exe` 用內建 HTTP client，唔使 curl。

```bash
python3 paste_curl_download.py --cli -f "2026-09-04 微積分 L1"
```

## 注意

- Zoom signed URL 會過期；cookie 綁瀏覽器同 IP
- 貼上內容含 session cookie，唔好分享
- 長片通常遠大於 5 MB
