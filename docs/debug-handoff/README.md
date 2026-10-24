# DEBUG HANDOFF — lỗi do phiên điều phối + debug tìm ra, gửi phiên BUILD

Phiên debug chạy và trace EAIS ở một mốc cố định, **không sửa code**. Thấy lỗi thì dừng ở chẩn đoán và ghi một tệp ở đây. Phiên BUILD điều tra lại, quyết cách sửa theo `CLAUDE.md`, rồi điền mục "Trạng thái BUILD" ở cuối tệp (quy ước ở `docs/PROMPT-BUILD-EAIS.md` §2).

- Tên tệp: `YYYY-MM-DD-NN-<tóm-tắt>.md`.
- Độ tin: CONFIRMED (đọc code + có vết đo) · STRONG HYPOTHESIS · HYPOTHESIS · UNKNOWN.
- Một handoff = một lỗi. Không gộp nhiều lỗi vào một tệp.

| Tệp | Lỗi | Mức | Trạng thái BUILD |
|---|---|---|---|
| `2026-10-21-01-trich-dan-thieu-so-khac.md` | 3 số "các số khác cùng kỳ" được S9 kiểm nhưng không có trong trích dẫn và bản ghi quyết định | THẤP | NHẬN (21/10 14:4x) |
| `2026-10-24-01-giao-nhau-rag-voi-nhanh-team.md` | 3 nhánh team chưa gộp (`m3/s4-vong-sua-2310`, `m1/vet-s8-s10`, `m1/5e-data-version`) xung đột với RAG khi rebase lên main — mọi khối là hai bên cùng thêm, giữ cả hai | chặn rebase | chờ người rebase |
