# ADR-0008 — Một bộ từ vựng nhóm lỗi (tập hợp lớn hơn, đảo được)

- **Ngày:** 20/10/2026, đêm.
- **Trạng thái:** **ĐÃ CHỐT TẠM.** Tự chốt theo D-OP-7: lựa chọn không ảnh hưởng benchmark / runtime thì chọn tập hợp lớn hơn đảo được, ghi ADR. Chủ dự án đảo được bất cứ lúc nào.
- **Liên quan:** D-OP-7 (`docs/CHO-QUYET-DINH.md`); `CLAUDE.md` P10; `docs/CONVENTIONS.md` §4.2; `docs/g7/HOP-DONG-G7.md` §8.

## Bối cảnh

Có ba danh sách nhóm lỗi, lệch nhau:

| Nguồn | Danh sách |
|---|---|
| `CLAUDE.md` P10 (16 nhóm) | DATA, PARSING, NORMALIZATION, ENTITY, SOURCE_AUTHORITY, ROUTER, RETRIEVAL, CONTEXT, MODEL, SKILL, TOOL, WORKFLOW, AUTH, TENANCY, VALIDATION, INFRASTRUCTURE |
| `CONVENTIONS` §4.2 cũ (18 nhóm) | DATA, PARSING, NORMALIZATION, ENTITY, ROUTER, RETRIEVAL, CONTEXT, MODEL, TOOL, SKILL, WORKFLOW, INTEGRATION, AUTH, TENANCY, CACHE, STREAM, VALIDATION, OBSERVABILITY |
| Phân loại tầng sai đầu tiên của G7 (prompt cố vấn, chủ dự án duyệt đêm 20/10) | DATA, NORMALIZATION, ENTITY, INTENT, SEMANTICS, SOURCE_AUTHORITY, PERMISSION, RETRIEVAL, CONTEXT, ROUTING, MODEL, CLAIM_BINDING, VALIDATION, SCORER, PROVIDER, UNKNOWN |

**Đo ảnh hưởng (20/10 đêm):**
- không mã Go, tệp YAML hay bộ ca nào dùng các danh sách này (`grep` toàn repo theo tên trường `failure_category` và các nhãn riêng);
- định dạng ca (`CONVENTIONS` §5) dùng trường `category` với từ vựng khác;
- nhãn `fault` trong mã là §4.3, tách riêng.

Các danh sách chỉ sống trong tài liệu.

## Quyết định

1. `CONVENTIONS` §4.2 thành **tập hợp của cả ba**. Tổng 27 nhóm: 18 nhóm cũ + SOURCE_AUTHORITY, INFRASTRUCTURE + INTENT, SEMANTICS, PERMISSION, ROUTING, CLAIM_BINDING, SCORER, PROVIDER, UNKNOWN.
2. **Ranh giới các cặp dễ nhầm** ghi ngay ở §4.2:
   - ROUTER: lỗi của thành phần; ROUTING: chọn sai đường cho một ca.
   - AUTH: cơ chế xác thực / phân quyền hỏng; PERMISSION: câu trả lời lộ dữ liệu ngoài quyền người hỏi; TENANCY: vượt ranh giới tenant.
   - MODEL: model sai khi ngữ cảnh đủ đúng; PROVIDER: API lỗi, hết hạn mức, quá giờ.
   - INFRASTRUCTURE: hạ tầng của ta.
   - SCORER: chính bộ chấm sai.
   - UNKNOWN: phải liệt kê riêng, không gộp.
3. **Không sửa `CLAUDE.md`.** P10 nay là tập con của §4.2, nên hết mâu thuẫn mà không đụng tệp của chủ dự án.

## Hệ quả

- Tài liệu, báo cáo, phân loại ca hỏng dùng nhãn nào trong §4.2 cũng được. Thêm nhãn mới thì cần ADR.
- Muốn thu hẹp về một danh sách ngắn hơn: chủ dự án quyết, rồi sửa §4.2 và các tài liệu dùng nhãn bị bỏ.
