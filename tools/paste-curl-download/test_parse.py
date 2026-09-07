#!/usr/bin/env python3
import tempfile
import unittest
from pathlib import Path

from paste_curl_download import (
    ParseError,
    filename_from_url,
    lecture_filename,
    parse_curl_paste,
    write_curl_config,
)

BASH = r"""
curl --url 'https://ssrweb.zoom.us/replay02/2026/09/04/GMT20260904-005633_Recording_1920x1080.mp4?data=abc&tid=v=2.0;clid=aw1' \
  -H 'Accept: */*' \
  -H 'Range: bytes=0-4128767' \
  -H 'Referer: https://hkmu.zoom.us/' \
  -H 'sec-ch-ua: "Chromium";v="152", "Not?A_Brand";v="24"' \
  -b '_zm_ssid=aw1_c_test; cf_clearance=abc|1|0|def'
"""

CMD = r"""
curl "https://example.com/files/lecture.mp4?x=1" ^
  -H "Range: bytes=100-200" ^
  -H "Referer: https://hkmu.zoom.us/" ^
  -b "sid=1"
"""

PREFIXED = r"""
~/Downloads && curl -L --fail -o GMT20260904-005633_Recording_1920x1080.mp4 --url 'https://ssrweb.zoom.us/a.mp4?q=1' \
  -H 'Range: bytes=0-99'
"""


CHROME_REAL = r"""
curl --url 'https://ssrweb.zoom.us/replay02/2026/09/04/1203006F-1514-4E01-951A-BE706105DB4F/GMT20260904-005633_Recording_1920x1080.mp4?response-content-type=video%2Fmp4&tid=v=2.0;clid=aw1&Key-Pair-Id=K1YYCGW8V4AHXW' \
  -H 'Accept: */*' \
  -H 'Connection: keep-alive' \
  -b '_zm_ssid=aw1_c_test; cf_clearance=abc|1|0|def' \
  -H 'Range: bytes=0-' \
  -H 'Referer: https://hkmu.zoom.us/' \
  -H 'User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/152.0.0.0 Safari/537.36' \
  -H 'sec-ch-ua: "Chromium";v="152", "Not?A_Brand";v="24", "Google Chrome";v="152"' \
  -H 'sec-ch-ua-mobile: ?0' \
  -H 'sec-ch-ua-platform: "Windows"'
"""


class ParseTests(unittest.TestCase):
    def test_chrome_real_layout(self):
        p = parse_curl_paste(CHROME_REAL)
        self.assertEqual(p.filename, "GMT20260904-005633_Recording_1920x1080.mp4")
        self.assertIn("Key-Pair-Id=K1YYCGW8V4AHXW", p.url)
        self.assertIn("tid=v=2.0;clid=aw1", p.url)
        self.assertEqual(p.cookie, "_zm_ssid=aw1_c_test; cf_clearance=abc|1|0|def")
        self.assertIn("Range: bytes=0-", p.headers)
        self.assertTrue(any('Google Chrome' in h for h in p.headers))

    def test_bash_zoom_style(self):
        p = parse_curl_paste(BASH)
        self.assertTrue(p.url.startswith("https://ssrweb.zoom.us/"))
        self.assertIn("tid=v=2.0;clid=aw1", p.url)
        self.assertEqual(p.filename, "GMT20260904-005633_Recording_1920x1080.mp4")
        self.assertEqual(p.cookie, "_zm_ssid=aw1_c_test; cf_clearance=abc|1|0|def")
        ranges = [h for h in p.headers if h.lower().startswith("range:")]
        self.assertEqual(ranges, ["Range: bytes=0-"])
        self.assertTrue(any(h.lower().startswith("referer:") for h in p.headers))
        ua = [h for h in p.headers if h.lower().startswith("sec-ch-ua:")]
        self.assertEqual(len(ua), 1)
        self.assertIn("Chromium", ua[0])

    def test_cmd_copy(self):
        p = parse_curl_paste(CMD)
        self.assertEqual(p.filename, "lecture.mp4")
        self.assertEqual(p.cookie, "sid=1")
        self.assertIn("Range: bytes=0-", p.headers)

    def test_strips_directory_prefix(self):
        p = parse_curl_paste(PREFIXED)
        self.assertTrue(p.url.endswith("a.mp4?q=1") or "a.mp4" in p.url)
        self.assertEqual(p.filename, "a.mp4")

    def test_empty(self):
        with self.assertRaises(ParseError):
            parse_curl_paste("   ")

    def test_rejects_non_https(self):
        with self.assertRaises(ParseError):
            parse_curl_paste("curl http://evil.example/x.mp4")

    def test_lecture_filename(self):
        self.assertEqual(lecture_filename("2026-09-04 微積分 L1", "GMT.mp4"), "2026-09-04 微積分 L1.mp4")
        self.assertEqual(lecture_filename("notes.mkv", "GMT.mp4"), "notes.mkv")
        self.assertEqual(lecture_filename("", "GMT20260904.mp4"), "GMT20260904.mp4")
        self.assertEqual(lecture_filename(r"C:\evil\a<>b", "x.mp4"), "a__b.mp4")
        with self.assertRaises(ParseError):
            lecture_filename("", "")
        self.assertEqual(
            filename_from_url("https://x.example/a/b/hello world.mp4"),
            "hello world.mp4",
        )

    def test_config_roundtrip(self):
        p = parse_curl_paste(BASH)
        with tempfile.TemporaryDirectory() as tmp:
            dest = Path(tmp) / p.filename
            conf = Path(tmp) / "curl.conf"
            write_curl_config(p, dest, conf)
            body = conf.read_text(encoding="utf-8")
            self.assertIn("location", body)
            self.assertIn('continue-at = "-"', body)
            self.assertIn("Range: bytes=0-", body)
            self.assertIn("_zm_ssid=aw1_c_test", body)
            self.assertNotIn("bytes=0-4128767", body)


if __name__ == "__main__":
    unittest.main()
