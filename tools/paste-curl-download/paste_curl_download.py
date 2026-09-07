#!/usr/bin/env python3
"""Paste Chrome 'Copy as cURL (bash)' and download the file.

Parses URL / headers / cookies only — never evals the pasted shell text.
Rewrites Range to bytes=0- so video 206 chunks become a full download.
"""

from __future__ import annotations

import argparse
import os
import re
import shutil
import subprocess
import sys
import tempfile
import threading
from dataclasses import dataclass, field
from pathlib import Path
from urllib.parse import unquote, urlparse

HEADER_RE = re.compile(
    r"""(?:-H|--header)\s+(?P<q>['"])(?P<val>.*?)(?P=q)""",
    re.IGNORECASE | re.DOTALL,
)
COOKIE_RE = re.compile(
    r"""(?:-b|--cookie)\s+(?P<q>['"])(?P<val>.*?)(?P=q)""",
    re.IGNORECASE | re.DOTALL,
)
URL_FLAG_RE = re.compile(
    r"""--url\s+(?P<q>['"])(?P<url>https://.+?)(?P=q)""",
    re.IGNORECASE | re.DOTALL,
)
URL_CURL_RE = re.compile(
    r"""\bcurl(?:\.exe)?\s+(?:-[A-Za-z]\s+)*['"](?P<url>https://[^'"]+)['"]""",
    re.IGNORECASE,
)
BARE_HTTPS_RE = re.compile(r"https://[^\s'\"\\]+")


@dataclass
class ParsedCurl:
    url: str
    headers: list[str] = field(default_factory=list)
    cookie: str = ""
    filename: str = "download.bin"


class ParseError(ValueError):
    pass


def default_download_dir() -> Path:
    home = Path.home()
    downloads = home / "Downloads"
    return downloads if downloads.is_dir() else home


def normalize_paste(text: str) -> str:
    text = text.replace("\r\n", "\n").replace("\r", "\n").strip()
    text = re.sub(r"(?m)^\$\s*", "", text)
    text = re.sub(r"^(?:cd\s+\S+\s*&&\s*)+", "", text)
    text = re.sub(r"^(?:~/[^\s]*|/[^\s]+|[A-Za-z]:\\[^\s]+)\s*&&\s*", "", text)
    text = re.sub(r"\\\s*\n", " ", text)
    text = re.sub(r"\^\s*\n", " ", text)
    return text.strip()


def filename_from_url(url: str) -> str:
    path = unquote(urlparse(url).path)
    name = path.rsplit("/", 1)[-1] if path else ""
    name = name.strip() or "download.bin"
    name = re.sub(r'[<>:"/\\|?*\x00-\x1f]', "_", name)
    if name in {".", ".."}:
        name = "download.bin"
    return name


def _dedupe_headers(headers: list[str]) -> list[str]:
    seen: set[str] = set()
    out: list[str] = []
    for raw in headers:
        key = raw.split(":", 1)[0].strip().lower()
        if key in seen:
            continue
        seen.add(key)
        out.append(raw.strip())
    return out


def parse_curl_paste(text: str) -> ParsedCurl:
    blob = normalize_paste(text)
    if not blob:
        raise ParseError("Empty paste.")

    url = ""
    flag = URL_FLAG_RE.search(blob)
    if flag:
        url = flag.group("url")
    else:
        curl_url = URL_CURL_RE.search(blob)
        if curl_url:
            url = curl_url.group("url")
        else:
            https = BARE_HTTPS_RE.findall(blob)
            if https:
                url = max(https, key=len)

    url = url.strip().rstrip("\\")
    if not url.startswith("https://"):
        raise ParseError("No https URL found. Paste Chrome Copy as cURL (bash).")

    headers = [m.group("val").strip() for m in HEADER_RE.finditer(blob)]
    cookies = [m.group("val").strip() for m in COOKIE_RE.finditer(blob)]
    cookie = cookies[-1] if cookies else ""

    rewritten: list[str] = []
    has_range = False
    for h in headers:
        name, _, value = h.partition(":")
        lname = name.strip().lower()
        if lname == "range":
            rewritten.append("Range: bytes=0-")
            has_range = True
        elif lname == "cookie" and not cookie:
            cookie = value.strip()
            rewritten.append(h)
        else:
            rewritten.append(h)
    if not has_range:
        rewritten.append("Range: bytes=0-")

    return ParsedCurl(
        url=url,
        headers=_dedupe_headers(rewritten),
        cookie=cookie,
        filename=filename_from_url(url),
    )


def _curl_config_escape(value: str) -> str:
    return value.replace("\\", "\\\\").replace('"', '\\"')


def write_curl_config(parsed: ParsedCurl, dest: Path, config_path: Path) -> None:
    lines = [
        "location",
        "fail",
        'retry = "3"',
        'continue-at = "-"',
        f'output = "{_curl_config_escape(str(dest))}"',
        f'url = "{_curl_config_escape(parsed.url)}"',
    ]
    if parsed.cookie:
        lines.append(f'cookie = "{_curl_config_escape(parsed.cookie)}"')
    for header in parsed.headers:
        lines.append(f'header = "{_curl_config_escape(header)}"')
    config_path.write_text("\n".join(lines) + "\n", encoding="utf-8")


def find_curl() -> str:
    env = os.environ.get("CURL_BIN")
    if env:
        return env
    for name in ("curl.exe", "curl"):
        path = shutil.which(name)
        if path:
            # Windows PowerShell aliases curl -> Invoke-WebRequest; which()
            # should still return curl.exe when present.
            if os.name == "nt" and path.lower().endswith("curl") and not path.lower().endswith("curl.exe"):
                continue
            return path
    raise FileNotFoundError("curl not found. Install Git for Windows or use Windows 10+ curl.exe.")


def download(parsed: ParsedCurl, dest: Path, curl_bin: str | None = None) -> int:
    dest.parent.mkdir(parents=True, exist_ok=True)
    curl = curl_bin or find_curl()
    with tempfile.TemporaryDirectory(prefix="paste-curl-") as tmp:
        config_path = Path(tmp) / "curl.conf"
        write_curl_config(parsed, dest, config_path)
        proc = subprocess.run(
            [curl, "-K", str(config_path)],
            check=False,
        )
        return proc.returncode


def run_cli(args: argparse.Namespace) -> int:
    if args.paste_file:
        text = Path(args.paste_file).read_text(encoding="utf-8")
    else:
        print("Paste Copy as cURL (bash), then Ctrl+D (Git Bash) / Ctrl+Z Enter (cmd).", file=sys.stderr)
        text = sys.stdin.read()
    try:
        parsed = parse_curl_paste(text)
    except ParseError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 2

    dest_dir = Path(args.output_dir).expanduser() if args.output_dir else default_download_dir()
    filename = args.filename or parsed.filename
    dest = dest_dir / filename
    print(f"URL:  {parsed.url.split('?', 1)[0]}", file=sys.stderr)
    print(f"File: {dest}", file=sys.stderr)
    try:
        code = download(parsed, dest)
    except FileNotFoundError as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 127
    if code != 0:
        print(f"curl exited {code}", file=sys.stderr)
        return code
    print(f"saved {dest} ({dest.stat().st_size} bytes)", file=sys.stderr)
    return 0


def run_gui() -> int:
    try:
        import tkinter as tk
        from tkinter import filedialog, messagebox, ttk
    except ImportError:
        print("tkinter not available; use --cli", file=sys.stderr)
        return 1

    root = tk.Tk()
    root.title("Paste cURL Download")
    root.minsize(720, 560)
    root.geometry("880x640")

    pad = {"padx": 12, "pady": 6}

    ttk.Label(
        root,
        text="Paste Chrome → Copy as cURL (bash). Range is rewritten to bytes=0- (full file).",
        wraplength=840,
    ).pack(anchor="w", **pad)

    text = tk.Text(root, wrap="none", height=16, undo=True)
    text.pack(fill="both", expand=True, padx=12, pady=4)

    opts = ttk.Frame(root)
    opts.pack(fill="x", **pad)

    ttk.Label(opts, text="Save to").grid(row=0, column=0, sticky="w")
    dir_var = tk.StringVar(value=str(default_download_dir()))
    dir_entry = ttk.Entry(opts, textvariable=dir_var, width=60)
    dir_entry.grid(row=0, column=1, sticky="ew", padx=6)

    def browse() -> None:
        chosen = filedialog.askdirectory(initialdir=dir_var.get() or str(default_download_dir()))
        if chosen:
            dir_var.set(chosen)

    ttk.Button(opts, text="Browse", command=browse).grid(row=0, column=2)

    ttk.Label(opts, text="Filename").grid(row=1, column=0, sticky="w", pady=(8, 0))
    name_var = tk.StringVar()
    ttk.Entry(opts, textvariable=name_var, width=60).grid(
        row=1, column=1, sticky="ew", padx=6, pady=(8, 0)
    )
    opts.columnconfigure(1, weight=1)

    log = tk.Text(root, height=8, wrap="word", state="disabled")
    log.pack(fill="both", expand=False, padx=12, pady=4)

    def append_log(msg: str) -> None:
        log.configure(state="normal")
        log.insert("end", msg + "\n")
        log.see("end")
        log.configure(state="disabled")
        root.update_idletasks()

    def on_paste_change(_event=None) -> None:
        raw = text.get("1.0", "end")
        if "https://" not in raw:
            return
        try:
            parsed = parse_curl_paste(raw)
        except ParseError:
            return
        if not name_var.get().strip():
            name_var.set(parsed.filename)

    text.bind("<KeyRelease>", on_paste_change)

    def do_download() -> None:
        raw = text.get("1.0", "end")
        try:
            parsed = parse_curl_paste(raw)
        except ParseError as exc:
            messagebox.showerror("Parse error", str(exc))
            return
        filename = name_var.get().strip() or parsed.filename
        dest = Path(dir_var.get()).expanduser() / filename
        append_log(f"Downloading → {dest}")
        append_log(parsed.url.split("?", 1)[0])
        dl_btn.configure(state="disabled")

        def work() -> None:
            err: Exception | None
            try:
                code = download(parsed, dest)
                err = None
            except Exception as exc:  # surface to UI thread
                code = -1
                err = exc

            def finish() -> None:
                dl_btn.configure(state="normal")
                if err is not None:
                    append_log(str(err))
                    messagebox.showerror("Download failed", str(err))
                    return
                if code != 0:
                    append_log(f"curl exited {code}")
                    messagebox.showerror("Download failed", f"curl exited {code}")
                    return
                size = dest.stat().st_size
                append_log(f"Saved {size} bytes")
                if size < 5_000_000:
                    append_log(
                        "Warning: file < 5 MB. If this was a long Zoom MP4, Range/URL may still be wrong."
                    )
                messagebox.showinfo("Done", f"Saved:\n{dest}\n{size} bytes")

            root.after(0, finish)

        threading.Thread(target=work, daemon=True).start()

    btns = ttk.Frame(root)
    btns.pack(fill="x", **pad)
    dl_btn = ttk.Button(btns, text="Download", command=do_download)
    dl_btn.pack(side="left")
    ttk.Button(btns, text="Clear", command=lambda: text.delete("1.0", "end")).pack(side="left", padx=8)

    root.mainloop()
    return 0


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Download from a pasted Chrome cURL command.")
    parser.add_argument("--cli", action="store_true", help="Read paste from stdin / --paste-file")
    parser.add_argument("--paste-file", help="File containing the copied cURL command")
    parser.add_argument("-o", "--output-dir", help="Directory to save into (default: Downloads)")
    parser.add_argument("-f", "--filename", help="Override output filename")
    args = parser.parse_args(argv)

    want_gui = not args.cli and not args.paste_file and sys.stdin.isatty()
    if want_gui:
        return run_gui()
    return run_cli(args)


if __name__ == "__main__":
    sys.exit(main())
