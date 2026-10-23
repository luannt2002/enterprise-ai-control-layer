# Dòng S7 — bộ đo, số phát triển, và bộ ca độc lập

> **Viết cho:**
> - chủ dự án — để đọc số và quyết;
> - người soạn bộ ca độc lập — mục 3;
> - team BUILD — để biết dòng S7 đo bằng gì.
>
> **Ngày:** 23/10/2026. **Nhánh:** `rag/0-viec`.

Dòng S7 của CỔNG TRƯỚC AI (`docs/bao-cao/CONG-TRUOC-AI.md`) yêu cầu ba vế:
- đoạn chứa đáp án nằm trong top-3 ≥ 90%;
- 0 đoạn ngoài quyền;
- 100% bằng chứng có nguồn gốc.

Theo khuyến nghị (a) của D-AI-03, S7 nằm ngoài MVP và được đo ở nhánh RAG.

## 1. Bộ đo

`evals/rag/s7_test.go` gồm ba test:

| Test | Làm gì |
|---|---|
| `TestS7BoPhatTrien` | đo trên chỉ mục trong bộ nhớ |
| `TestS7BoPhatTrienQuaPG` | đo trên đường PostgreSQL của ứng dụng, cùng bộ câu; phải trúng đúng những câu mà đường bộ nhớ trúng |
| `TestS7PhepDoPhanBiet` | chứng minh bộ đo phân biệt được đúng sai (mô tả dưới) |

`TestS7PhepDoPhanBiet` kiểm rằng:
- bộ đếm đỏ khi chỉ mục bỏ ACL, khi chỉ mục bỏ phạm vi, khi trả bản nháp, khi đoạn thiếu nguồn;
- bộ kiểm chặn đáp án không có trong kho.

Hai nguyên tắc của bộ đo:
- **Không tự chấm.** Quyền đọc và "làm căn cứ được" (trạng thái, hiệu lực, bản duy nhất) được tính lại từ ACL, phạm vi và policy của pack, không gọi mã của gói `rag`. Năm đột biến cố ý cài vào bộ đo (bỏ ACL, bỏ phạm vi, bỏ kiểm nguồn, bỏ kiểm đáp án, IDF tính cả tài liệu không thấy) đều bị bắt.
- **Kết luận theo luật cố định.** Dòng log kết thúc bằng `dòng S7: ĐẠT | KHÔNG ĐẠT`, theo luật: trúng / số câu trả lời được ≥ 90%, và 0 ngoài quyền, 0 không làm căn cứ, 0 thiếu nguồn.

## 2. Bộ phát triển và số đo — KHÔNG phải số của cổng

Bộ phát triển do chính người làm RAG soạn, nên số đo dưới đây chỉ là số phát triển.

**Kho** `tenants/construction/knowledge/s7/kho/`:
- 15 tài liệu (17 bản), 138 dòng, giả lập theo `docs/tenants/construction/01-business-map.md` §7 (BR-FUEL, BR-RPT, BR-MNT, BR-PROC, BR-PAY, BR-ACC, BR-SAF, BR-CHG, BR-WH, BR-DOC). Các mốc chi tiết như 250 giờ bảo dưỡng, 50 cm mỗi lớp bê tông là giả lập bổ sung.
- Có tài liệu gây nhiễu: "24 giờ" và "48 giờ" xuất hiện ở ba quy trình khác nhau.
- Có một bản nháp, có tài liệu theo phạm vi PRJ-A và PRJ-B, có tài liệu chỉ vài vai được đọc.

**Bộ câu** `tenants/construction/knowledge/s7/phat-trien.yaml`: 41 câu, gồm 36 câu có đáp án và 5 câu bẫy (2 ngoài quyền, 1 ngoài phạm vi, 2 không có tài liệu).

**Băm ghi lúc 23/10 10:35, trước lần chạy đầu:**
- `phat-trien.yaml`: `bb6b2dc8911891630fc6830658ef9fc08da506a5e3b8f51c6315ee7e2312aa08`
- `kho/` (18 tệp, `sha256sum` theo tên rồi băm lại): `2ab7ece9b0d98e06eac960f99bf304810b06d23272afc54a9c542ea2513391b4`

Sau lần chạy đầu, chỉ thêm dòng `truot_biet_truoc`; không sửa câu hỏi, đáp án hay kho. Kiểm lại bằng lệnh:
`grep -v '^    truot_biet_truoc: ' tenants/construction/knowledge/s7/phat-trien.yaml | sha256sum` phải ra `bb6b2dc8…`.

| Lần | Bộ nhớ | PG | Ghi chú |
|---|---|---|---|
| 1 (trước khi sửa) | 31/36 = 86,1% | 30/36 = 83,3% | S7D-25: bộ nhớ trúng, PG trượt. Nguyên nhân: IDF đường bộ nhớ tính trên mọi tài liệu của tenant, đường PG chỉ trên tài liệu qua ACL. Điểm số phụ thuộc cả tài liệu người hỏi không được đọc |
| 2 (sau khi sửa: IDF chỉ trên đoạn người hỏi thấy) | 30/36 = 83,3% | 30/36 = 83,3% | hai đường khớp từng câu |

Ở lần 2, cả hai đường đều đạt: 0 đoạn ngoài quyền, 0 đoạn không làm căn cứ, 0 thiếu nguồn.

**Dòng S7 trên bộ phát triển: KHÔNG ĐẠT, vì 83,3% < 90%.**

**Cách đọc số:**
- **6 ca trượt cùng một nguyên nhân:** người hỏi dùng chữ khác tài liệu, ví dụ "tai nạn" ≠ "sự cố", "trả tiền" ≠ "thanh toán", "trễ lịch" ≠ "quá hạn", "phần" ≠ "mục". Bộ tìm từ khoá không biết đồng nghĩa. Đây là số đo đầu tiên có thể làm căn cứ cho cổng mở "đồng nghĩa của pack" hoặc "embedding" (CLAUDE.md §19). Chỉ mở cổng khi bộ độc lập ở mục 3 cho kết quả cùng chiều.
- **5/5 câu không có đáp án vẫn bị trả đoạn.** `MinScore` = 2,0 quá dễ dãi để biết khi nào nên im. Vế này không thuộc S7, nhưng nó đưa đoạn lạc đề tới model. Xem mục "Biết im" ngay dưới.

### Biết im — P1 (23/10): đã sửa phần lớn, còn 2/5

**Lỗi:** đi trọn pipeline, câu kho không trả lời được ra **RESOLVED với đoạn lạc đề**.
- "thủ tục xin giấy phép xây dựng" → đoạn đổ bê tông trụ (khớp "tục" trong "liên tục");
- "quy trình xin nghỉ phép năm" → "báo cáo ngày phải đủ năm mục".

Câu trả lời chép nguyên văn tài liệu nên qua bước kiểm: P5 bị phá âm thầm. Lỗi được ghim bằng `TestRAGKhongTraLoiBangDoanLacDe` (nay là kiểm thường, đã xanh — xem luật 2').

**Ba luật đã thử, mỗi luật chốt trước khi đo. Luật 1 và 2 trượt, đã hoàn tác; luật 2' được giữ:**

| Luật | Câu không có đáp án mà trả đoạn | Top-3 bộ phát triển | Test sẵn có |
|---|---|---|---|
| hiện tại: chỉ `MinScore` 2,0 | 5/5 | 30/36 | xanh |
| 1. đoạn phải phủ ≥ 50% IDF của câu hỏi (từ ngoài kho = IDF lớn nhất) | 0/5 | **19/36** | vỡ CON-T-004 và 2 test |
| 2. im khi ≥ 1/2 từ nội dung của câu nằm ngoài kho người hỏi thấy | chưa đo (dừng ở test vỡ) | chưa đo | vỡ 5 test (kho nhỏ; câu có mã máy; phần số liệu của HYBRID) |
| **2'. luật 2 xét trên CHỦ ĐỀ:** câu gốc bỏ các span số / lượng / tiền / ngày / kỳ / giờ / mã của S2, giữ thuật ngữ (`pipeline.KnowledgeTopic` → `KnowledgeQuery.Topic`) | **2/5** | **30/36** (không mất câu nào) | xanh; sửa dữ liệu thử của một test team (mục dưới) |

**Nguyên nhân chung:** từ dùng để hỏi ("mấy tiếng", "dùng"), mã thực thể và phần số liệu nằm chung trong câu hỏi; từ khoá không tách được "điều được hỏi" khỏi "cách hỏi". Cứu bằng danh sách từ loại trừ là lặp lỗi danh sách đóng (ADR-0010), nên không làm.

**Luật 2' — cách đọc:**
- Luật 2' được thiết kế SAU khi thấy luật 1, 2 trượt trên chính bộ phát triển, nên số phát triển của nó lạc quan; chỉ bộ độc lập (§3) quyết.
- Hệ quả đã biết: tài liệu không nêu chủ đề được hỏi thì không còn được dùng, kể cả khi một hàng trong nó "có vẻ" liên quan. Test `TestLuongTaiLieuTrangTriVaNhanDonViKhongLot` của team dùng hàng trần `| Máy xúc | 4.000 | lít/ca |` cho câu "Tài liệu định mức cấp phát…" — nay dừng ở truy xuất (INSUFFICIENT_EVIDENCE, không lộ số) nên không tới lớp kiểm mà test canh; dữ liệu thử đổi thành `| Định mức cấp phát máy xúc | 4.000 | lít/ca |` để test vẫn canh đúng lớp kiểm (kiểm của test giữ nguyên).
- Còn 2/5 câu `khong_co` nhận đoạn: S7D-38, S7D-39, cả hai là bẫy `ngoai_quyen` — đáp án ở tài liệu người hỏi không được đọc (không lộ: 0 đoạn ngoài quyền), nhưng chủ đề trùng một phần chữ với tài liệu họ được đọc ("bị nạn" ở quy trình sự cố, "phiếu" ở quy trình cấp dầu) nên vẫn nhận đoạn lạc đề. 5 đột biến (bỏ chủ đề, giữ số / mã trong chủ đề, bỏ cả thuật ngữ, `>` thay `≥`, bỏ luật) đều bị bắt.

**Nấc tiếp theo nếu bộ độc lập vẫn thấy câu lạc đề:**
- tín hiệu ngữ nghĩa (reranker / embedding) qua cổng CLAUDE.md §19. Bằng chứng đã có cho cổng: 2/2 câu lạc đề ra RESOLVED, 5/5 câu `khong_co` nhận đoạn, 6/36 câu trượt vì đồng nghĩa.

**Phép so ba hệ trên bộ này** (`TestSoSanhBaHeTrenS7`, cùng câu, cùng kho, cùng bộ đếm; bộ do người làm soạn ⇒ minh hoạ, không phải thống kê):

| Hệ | Top-3 trúng | Đoạn ngoài quyền | Đoạn không làm căn cứ (nháp / hết hiệu lực / xung đột) | Câu không có đáp án mà trả đoạn |
|---|---|---|---|---|
| EAIS (lượt hỏi thật) | 30/36 | 0 | 0 | 2/5 |
| RAG thường (tenant + ACL vai; không phiên bản / hiệu lực / phạm vi / chủ đề) | 25/36 | 4 | 19 | 5/5 |
| Nạp thẳng (mọi tài liệu vai được đọc) | — | 25 | 64 | 5/5 |

**Lệnh chạy:**
```
go test -count=1 -v -run 'TestS7' ./evals/rag/
```
Đường PG cần `EAIS_PG_ADMIN_DSN` và `EAIS_PG_DSN`, như mọi test của pgstore.

## 3. Bộ ca độc lập — số của cổng

**Người soạn:**
- KHÔNG phải phiên RAG và không phải agent đang sửa `intelligence/rag`.
- Không đọc `phat-trien.yaml`, không đọc mã của gói `rag`, không chạy hệ trước khi đóng băng.
- Chỉ đọc `kho/` và `docs/tenants/construction/01-business-map.md`.

**Nội dung:**
- ≥ 50 câu `tra_loi`. Với 50 câu, mỗi câu trượt là 2%, nên ngưỡng 90% mới có nghĩa.
- ≥ 10 câu `khong_co`, trong đó ≥ 3 `ngoai_quyen`, ≥ 3 `ngoai_pham_vi`.
- Trải đủ các vai trong `kho/acl.json`.
- Có câu gõ không dấu, có viết tắt (PXK, PO, TVGS, CĐT), có câu hỏi ở ngày khác để đổi bản hiệu lực.
- Viết như người ở công trường hỏi. Không chép chữ của tài liệu và không né chữ của tài liệu.

**Kho:**
- Dùng `kho/` hiện có.
- Được thêm tài liệu gây nhiễu hoặc tài liệu mới vào **bản sao** kho đặt cạnh tệp câu hỏi. Kho của bộ độc lập là thư mục `kho/` nằm cạnh tệp câu hỏi đó.

**Khuôn tệp:**
- Đúng khuôn của `phat-trien.yaml`: cùng tên trường; khoá lạ thì bộ đo từ chối.
- `chua` là chuỗi con nguyên văn của đoạn chứa đáp án.
- Không có `truot_biet_truoc`.

**Đóng băng** theo quy ước ADR-0006 §10: chỉ commit băm và số đếm, nội dung giữ ở máy tới lúc chạy.
```
cd <thư mục bộ>
{ sha256sum doc-lap.yaml; (cd kho && ls | sort | xargs sha256sum); } > ../s7-manifest.txt
```
Commit `s7-manifest.txt` cùng số câu theo `loai` và `bay` TRƯỚC lần chạy.

**Chạy đúng MỘT lần:**
```
EAIS_S7_BO=<thư mục bộ>/doc-lap.yaml go test -count=1 -v -run 'TestS7BoPhatTrien' ./evals/rag/
```
- Test đỏ ở những câu trượt là bình thường, vì bộ độc lập không có cờ. Số của cổng là dòng `dòng S7: …` trên cả hai đường.
- Sau khi chạy: so băm với manifest, công bố nội dung, ghi số vào `CONG-TRUOC-AI.md` dòng S7. Đây là tệp của team; gửi qua bàn giao.

### Trạng thái bộ độc lập (23/10)

- **Đã soạn và đóng băng 11:53**: 106 câu (92 `tra_loi`; 14 `khong_co` = 5 `ngoai_quyen`, 4 `ngoai_pham_vi`, 5 `khong_co_tai_lieu`), kho
  23 tệp (17 tệp gốc + 5 tài liệu nhiễu người soạn thêm + `acl.json`). Băm và số đếm: `s7-doc-lap-manifest.txt` (tổng `2759dd22…`).
- Người soạn: một agent riêng, chỉ đọc kho, `01-business-map.md`, `policy.yaml`; không đọc mã RAG, bộ phát triển; không chạy hệ. Phiên RAG
  không mở nội dung câu hỏi (đóng băng chỉ in băm và số đếm).
- Người soạn ghi rủi ro: các vai ngoài `policy.yaml` coi như không có scope "all"; bộ chấm đo TRUY XUẤT nên không xét capability
  `documents.read` (accountant và vai ngoài policy không có capability này ở pipeline — đúng: S7 là dòng của truy xuất).
- **Chưa chạy.** Chạy đúng một lần sau khi manifest được commit.

## 4. Còn mở

- Ai soạn bộ độc lập — chủ dự án chỉ định.
- Nếu bộ độc lập KHÔNG ĐẠT ở vế top-3 vì đồng nghĩa, nấc kế tiếp là:
  - chọn giữa từ điển đồng nghĩa trong pack (rẻ, tất định, có kiểm) và embedding (cần hợp đồng embedding U9, không gọi model trả tiền khi chưa được cấp);
  - so cả hai trên chính bộ phát triển trước, rồi chạy một bộ độc lập MỚI. Không chạy lại bộ cũ.
