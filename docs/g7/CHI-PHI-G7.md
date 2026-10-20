# G7 — số lời gọi model và ước tính chi phí

> Đo 20/10/2026 trên revision `85108c4`. **Chưa gọi model thật, chưa chi đồng nào.**
> Hợp đồng benchmark: `docs/g7/HOP-DONG-G7.md`.

## 1. Cách đo

Chạy khô đúng đường mã thật: `g7run` + client `models.Messages`, `EAIS_MODEL_BASE_URL` trỏ vào một máy chủ giả trên máy. Máy chủ giả ghi từng yêu cầu và trả nguyên văn đoạn bằng chứng đầu. Không có lời gọi nào ra ngoài.

Chạy lại được:
- script máy chủ giả: `scripts/g7-chay-kho.py`;
- lệnh: mục 6.

## 2. Số lời gọi (mục D của checkpoint)

| Hệ | Mỗi lượt | k = 3 | Nguồn số |
|---|---|---|---|
| EACL | **7** (chỉ ca route RAG / HYBRID tới bước sinh: 068, 092, 093, 094, 095, 096, 116) | **21** | **đo** |
| B2 — RAG đơn giản | **117** (1 / câu hỏi) | **351** | **đo** |
| B1 — LLM trực tiếp | 117 | 351 | theo thiết kế (chưa hiện thực) |
| **Tổng, chỉ B2** | | **372** | **đo** (máy chủ giả nhận đúng 372 yêu cầu) |
| **Tổng, B1 + B2** | | **723** | 372 đo + 351 theo thiết kế |
| Thử khói trước khi chạy | | ~3 | |

- Ba lượt gửi prompt **giống hệt nhau** (đo kích thước từng prompt).
- Ca 072 và 084 thuộc route RAG nhưng **không gọi model**: một ca không có đoạn tài liệu nào đạt ngưỡng; một ca hỏi lượng nghiệp vụ mà chỉ có tài liệu trả lời.
- Bộ giữ lại (D-G7-5, nếu duyệt): thêm khoảng 40 × 3 = 120 lời gọi cho mỗi baseline, EACL tuỳ số ca RAG / HYBRID.

## 3. Kích thước đầu vào (đo bằng ký tự Unicode)

| Hệ | Lời gọi | Ký tự vào (system + user) | Trung bình / lời gọi | Lớn nhất (user) | Đoạn bằng chứng TB / max |
|---|---|---|---|---|---|
| EACL | 21 | 17.454 | 831 | 554 | 2,7 / 3 |
| B2 | 351 | 449.898 | 1.282 | 1.222 | 8,4 / 10 |
| B1 | 351 | ước tính 6,2–9,7 triệu | — | — | — |

Cách ước tính B1:
- dữ liệu functional của tenant A có 2.739 bản ghi;
- cửa sổ 18–20/10 có 112 bản ghi;
- chữ thô dạng văn bản ≈ 180 KB, tức khoảng 66 ký tự / bản ghi;
- 110 ca hỏi theo ngày: mỗi prompt khoảng 7–19 nghìn ký tự;
- 7 ca hỏi theo tháng: khoảng 170 nghìn ký tự mỗi prompt.

B1 đắt gấp hơn 10 lần B2 vì phải nhồi dữ liệu thô.

## 4. Quy ra token (ƯỚC TÍNH)

Tài liệu Models overview: 1 triệu token ≈ 2,5 triệu ký tự Unicode với tokenizer hiện tại. Tiếng Việt có dấu có thể tốn token hơn, nên dùng khoảng **2,0–3,0 ký tự / token**.

| Phần | Token vào | Token ra (giả định) |
|---|---|---|
| EACL + B2 | 156–234 nghìn (giữa: 187 nghìn) | 22–74 nghìn (60–200 token / câu trả lời JSON) |
| B1 | 2,1–4,9 triệu (giữa: 2,5–3,9 triệu) | 21–70 nghìn |

Giả định:
- thinking tắt (Haiku 5.5: `disabled`) hoặc không thinking trước (Sonnet 5.5: `between_tools`), nên không có token thinking. **Sẽ kiểm bằng lượt thử khói**;
- trần cứng mỗi lời gọi: `max_tokens` = 1024.

Đếm token chính xác: dùng trường `usage` của lượt thử khói, hoặc endpoint đếm token của nhà cung cấp. Cả hai đều cần khoá, nên **CHƯA ĐO**.

## 5. Chi phí ước tính

Giá chính thức, tra 20/10/2026 ở `platform.claude.com/docs/en/about-claude/pricing`, USD / triệu token:
- Haiku 5.5: vào 0,10, ra 0,50 (prompt ≤ 100 nghìn token);
- Sonnet 5.5: vào 2, ra 10;
- Opus 5.5: vào 4, ra 20;
- Batch API giảm 50% cả vào lẫn ra.

| Model | EACL + B2 | B1 | **Tổng B1 + B2** | Trường hợp xấu nhất (mọi câu chạm 1.024 token ra) |
|---|---|---|---|---|
| `claude-haiku-5-5` | 0,03–0,06 USD | 0,2–0,5 USD | **≈ 0,3–0,6 USD** | ≈ 1 USD |
| `claude-sonnet-5-5` | 0,5–1,2 USD | 4,4–8,5 USD | **≈ 5–10 USD** | ≈ 15 USD |
| `claude-opus-5-5` | 1,1–2,4 USD | 8,8–17 USD | **≈ 10–20 USD** | ≈ 30 USD |

- Cộng 10% dự phòng cho thử lại và thử khói.
- Prompt dài nhất của B1 (ca theo tháng) ≈ 57–85 nghìn token, dưới ngưỡng 100 nghìn token tính giá cao của Haiku 5.5.

## 6. Giảm số lời gọi mà không làm yếu phép đo

| Cách | Tác động | Có làm yếu phép đo không |
|---|---|---|
| EACL không gọi model ở route tất định | đã có: 7 thay vì 117 mỗi lượt | không. Đây chính là luận điểm cần đo |
| Batch API | −50% tiền | không, nhưng mất độ trễ từng lời gọi. Đo độ trễ trên một lượt đồng bộ riêng (D-G7-3) |
| Cache phần dữ liệu thô của B1 (đặt dữ liệu trước, câu hỏi sau) | đọc cache tính 10% giá vào; giảm phần lớn tiền B1 vì nhiều ca chung cửa sổ ngày | không, nếu ghi rõ trong hợp đồng. Cần sửa cách dựng prompt của B1 |
| Giảm k từ 3 xuống 1 | −67% | **có**: mất trung vị, không thấy ca chập chờn. Không khuyên |
| Bỏ B1 | −49% lời gọi, −80–90% tiền | không làm yếu phép so với B2, nhưng kết luận cao nhất chỉ còn PARTIAL (mục 1 của hợp đồng) |

## 7. Lệnh chạy khô (đo lại số ở trên)

```bash
D=$(mktemp -d)
python3 -I scripts/g7-chay-kho.py 18080 $D/requests.jsonl &
go run ./tenants/construction/win/gen/cmd/gen -world tenants/construction/win/world.yaml -out $D/data -profile functional -seed 42
EAIS_MODEL_PROVIDER=anthropic EAIS_MODEL=chay-kho EAIS_MODEL_API_KEY=khong-phai-khoa-that \
EAIS_MODEL_BASE_URL=http://127.0.0.1:18080 go run ./tenants/construction/win/g7run -data $D/data -k 3
wc -l $D/requests.jsonl      # 372
kill %1
```
