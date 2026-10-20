#!/usr/bin/env python3
# Máy chủ giả kiểu Messages API cho CHẠY KHÔ G7 (docs/g7/CHI-PHI-G7.md): ghi lại mọi yêu cầu, trả lời bằng
# nguyên văn đoạn bằng chứng đầu. Dùng: python3 -I scripts/g7-chay-kho.py <cổng> <tệp log jsonl>.
# Chỉ dùng cho chạy khô đếm lời gọi / kích thước prompt — KHÔNG phải model, KHÔNG gọi mạng ngoài.
import json, re, sys
from http.server import BaseHTTPRequestHandler, HTTPServer
LOG = sys.argv[2]
class H(BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('content-length', 0))
        body = self.rfile.read(n)
        req = json.loads(body)
        user = req['messages'][0]['content']
        m = re.search(r'^\[([^\]]+)\] (.*)$', user, re.M)
        ans = {"answer": m.group(2) if m else "", "evidence_ids": [m.group(1)] if m else []}
        with open(LOG, 'a', encoding='utf-8') as f:
            f.write(json.dumps({"bytes": n, "system_chars": len(req['system']), "user_chars": len(user),
                                "evidence": len(re.findall(r'^\[[^\]]+\] ', user, re.M)),
                                "model": req['model'], "temperature": req.get('temperature'), "thinking": req.get('thinking'),
                                "output_config": req.get('output_config'), "max_tokens": req['max_tokens'],
                                "question": user.split('\n', 1)[0]}, ensure_ascii=False) + '\n')
        out = json.dumps({"content": [{"type": "text", "text": json.dumps(ans, ensure_ascii=False)}],
                          "usage": {"input_tokens": 0, "output_tokens": 0}}).encode()
        self.send_response(200); self.send_header('content-type', 'application/json'); self.end_headers(); self.wfile.write(out)
    def log_message(self, *a): pass
HTTPServer(('127.0.0.1', int(sys.argv[1])), H).serve_forever()
