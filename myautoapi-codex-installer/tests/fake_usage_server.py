#!/usr/bin/env python3
"""Local stand-in for GET /v1/usage used by tests/installers.sh.

The answer depends on a word in the synthetic test key, so one server covers
every case. It prints its port, then serves until killed. Each request's
Authorization header is appended to the log file given as the argument.
"""

import http.server
import json
import sys
import time

ANSWERS = (
    ("rejected", 401, {"error": {"type": "authentication_error", "message": "Invalid API key"}}),
    ("forbidden", 403, {"error": {"message": "blocked"}}),
    ("disabled", 200, {"mode": "quota_limited", "isValid": False, "status": "disabled", "remaining": 5}),
    ("zero", 200, {"mode": "unrestricted", "isValid": True, "quota": {"remaining": 9}, "remaining": 0, "unit": "USD"}),
    ("noremaining", 200, {"mode": "unrestricted", "isValid": True, "planName": "Pro", "unit": "USD"}),
    ("broken", 500, {"error": "server"}),
    # /v1/usage skips billing checks: expired or used-up keys still say isValid true.
    ("statusexpired", 200, {"isValid": True, "mode": "quota_limited", "status": "expired"}),
    ("exhausted", 200, {"isValid": True, "mode": "quota_limited", "quota": {"limit": 5, "remaining": 0, "used": 5},
                        "remaining": 0, "status": "quota_exhausted", "unit": "USD"}),
    ("pastdate", 200, {"days_until_expiry": 0, "expires_at": "2001-02-03T04:05:06.789+08:00", "isValid": True,
                       "mode": "quota_limited", "status": "active"}),
    ("futuredate", 200, {"expires_at": "2999-01-01T00:00:00Z", "isValid": True, "mode": "quota_limited",
                         "rate_limits": [{"remaining": 0, "window": "5h"}], "remaining": 4, "status": "active", "unit": "USD"}),
)
USABLE = {"daily_usage": [{"remaining": 0}], "isValid": True, "mode": "unrestricted",
          "planName": "钱包余额", "quota": {"remaining": 0}, "remaining": 12.345, "unit": "USD"}


class Handler(http.server.BaseHTTPRequestHandler):
    def handle(self) -> None:
        # The timeout cases hang up before the slow answer is written.
        try:
            super().handle()
        except (BrokenPipeError, ConnectionResetError):
            pass

    def do_GET(self) -> None:
        authorization = self.headers.get("Authorization", "")
        with open(sys.argv[1], "a", encoding="utf-8") as log:
            log.write(f"{self.path} {authorization}\n")
        status, body = 404, {"error": "not found"}
        if self.path == "/v1/usage" and "slow" in authorization:
            # Longer than the shortened timeout the tests give the scripts.
            time.sleep(3)
        if self.path == "/v1/usage" and "notjson" in authorization:
            data = b"<html>login</html>"
            self.send_response(200)
            self.send_header("Content-Type", "text/html")
            self.send_header("Content-Length", str(len(data)))
            self.end_headers()
            self.wfile.write(data)
            return
        if self.path == "/v1/usage":
            status, body = 200, USABLE
            for word, answer_status, answer in ANSWERS:
                if word in authorization:
                    status, body = answer_status, answer
                    break
        data = json.dumps(body, ensure_ascii=False, separators=(",", ":"), sort_keys=True).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json; charset=utf-8")
        self.send_header("Content-Length", str(len(data)))
        self.end_headers()
        self.wfile.write(data)

    def log_message(self, *arguments) -> None:
        pass


server = http.server.ThreadingHTTPServer(("127.0.0.1", 0), Handler)
print(server.server_address[1], flush=True)
server.serve_forever()
