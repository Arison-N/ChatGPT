# Paste cURL Download

貼上 Chrome **Copy as cURL (bash)**，用你自己嘅**課堂檔名**儲存完整 MP4（自動把 `Range` 改成 `bytes=0-`）。

## Windows：雙擊 `.exe`

1. 開 `tools/paste-curl-download/dist/PasteCurlDownload.exe`（或雙擊 `Start.bat`）
2. **課堂檔名**填例如 `2026-09-04 微積分 L1`（未寫 `.mp4` 會自動加）
3. Chrome Network → MP4 → Copy as cURL (bash) → 貼上
4. 撳 **下載**

會用獨立 Chrome App 視窗；檔案預設去 `Downloads`。

重新編譯：

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
