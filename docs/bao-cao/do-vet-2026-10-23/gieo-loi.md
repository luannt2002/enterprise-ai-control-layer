# vet7g — mô tả lỗi gieo (phép đo "người soát chỉ đọc vết")

Nhãn: [ĐO] = tôi chạy lệnh ra số; [CODE] = đọc mã; [REPO] = lấy từ tệp của repo; [SUY] = suy luận, chưa đo.

Mã nền: origin/main = 086fff99025cfde4f9aed06b911180bb35235566 (có R2-03). Mọi binary (gốc và lỗi) dựng bằng
`go build -buildvcs=false -ldflags "-X main.buildVersion=086fff99025cfde4f9aed06b911180bb35235566"` để cột "mã" trong bảng vết GIỐNG HỆT
ở mọi thẻ; không ép thì binary dựng trong worktree mang `bin-sha256:<băm binary>` khác nhau từng binary và lộ thẻ nào đến từ binary lỗi [CODE: cmd/eais/version.go].
Pack và dữ liệu không bị sửa (phiên bản pack, dữ liệu trong bảng vết như nhau ở mọi thẻ). Lỗi chỉ nằm trong mã Go.

## Cách chấm một ca đỏ [ĐO]

1. Hỏi 474 câu của bốn bộ (117 + h1 + h3 + h4) qua server của từng binary (`POST /v1/ask`, token dev `dev-abc-infra-<principal>`, `-now world`), cùng PG `vet7g_db`.
2. Chấm câu trả lời bằng bản Python của `evals.RunWin` (route, outcome, value, contains, not_contains, candidates, conflict, model_calls từ usage của vết, cited_sources).
   Binary gốc: 423 ĐẠT / 51 ĐỎ; khớp 100% với danh sách ĐẠT / ĐỎ của cổng §6 (117: 117/117; h1: 91 + 13 đỏ; h3: 112 + 22 đỏ; h4: 103 + 16 đỏ).
   Hai lần chạy server gốc cho 0 bước khác và 0 câu trả lời khác (tất định).
3. Ca đỏ = ĐẠT với binary gốc, ĐỎ với binary lỗi. So `GET /v1/trace/{id}` (JSON) của cùng câu giữa hai binary theo TỪNG bước (trạng thái, vào, ra, quyết định, lý do, phương án loại,
   bằng chứng, lỗi; bỏ ms, mã vết): bước đầu tiên khác phải đúng là tầng gieo. Khác ở tầng nào thì loại và ghi tầng thấy được.

## Tám lỗi giữ lại (một lỗi mỗi tầng; binary `eais-<tầng>` — riêng S4 là `eais-s4f` vì thử nhiều biến thể)

Cột "giữ lại" = số ca đỏ có bước lệch đầu đúng tầng gieo / 423 ca ĐẠT đã hỏi.

| Tầng | Tệp:dòng (mã gốc) | Lỗi | Giữ lại | Thẻ |
|---|---|---|---|---|
| S2 | data/normalization/decimal.go:81 | Chấm đứng một mình theo sau đúng 3 chữ số bị đọc là thập phân kiểu Anh (`... && sep == ","`): "3.850 lít" thành 3,85 | 28 / 423 | 5 |
| S3 | data/entityresolution/resolver.go:96 | Bí danh khớp nhiều thực thể bị chọn ứng viên đầu thay vì báo mơ hồ (`len(cands) == 1` thành `>= 1`) | 32 / 423 | 5 |
| S4 | intelligence/intent/intent.go:479 | Đảo thứ tự ưu tiên ý định (`Priority >` thành `<`): ý định ưu tiên thấp (fuel_generic) thắng | 70 / 423 | 5 |
| S5 | intelligence/router/router.go:173 | `Policy.InScope` chỉ xét khoá của chính thực thể, không xét tổ tiên (`range lineage[:min(1, len(lineage))]`): kỹ sư dự án bị từ chối máy của dự án mình | 10 / 423 | 5 |
| S6 | core/decimal/decimal.go:100 (dùng ở data/sourceoftruth/query.go:138) | Biên dung sai: `Within` trả `c < 0` thay vì `c <= 0`: chênh đúng bằng dung sai bị coi là xung đột | 25 / 423 | 5 |
| S8 | core/pipeline/ask.go:569-570 | Nhánh thoát sớm của RAG ("câu đòi lượng nghiệp vụ mà chỉ có tài liệu") trả outcome OUT_OF_SCOPE thay vì INSUFFICIENT_EVIDENCE | 3 / 423 | 3 |
| S9 | reliability/validation/binding.go:73 | Ngày cuối của kỳ bị từ chối (`(ok && d == to)` thành `(!ok && d == to)`): câu trả lời nêu cả hai đầu kỳ bị "DATE_UNBOUND" | 14 / 423 | 5 |
| S10 | core/pipeline/ask.go:241 (chèn trước `d := a.decision(...)`) | Cắt câu trả lời ở 120 ký tự rune rồi thêm "…" | 10 / 423 | 5 |

Mã lỗi trong bảng ánh xạ: S2-dau-cham-nghin-doc-thanh-thap-phan, S3-bi-danh-trung-chon-ung-vien-dau, S4-uu-tien-y-dinh-dao-nguoc,
S5-pham-vi-chi-xet-may-khong-xet-du-an-cha, S6-dung-sai-bien-bo-dau-bang, S8-thoat-som-sai-outcome, S9-ngay-cuoi-ky-bi-tu-choi, S10-cat-cau-tra-loi-120-ky-tu.

Tổng: 38 thẻ đỏ (S8 chỉ 3) + 38 thẻ xanh = 76 thẻ.

## Ca đỏ bị loại vì bước lệch đầu KHÁC tầng gieo [ĐO]

- S2: 199 ca đỏ; 28 giữ; 169 lệch đầu ở S9 (câu hỏi không có số chấm nghìn nên S2 y hệt; câu TRẢ LỜI "3.850 L" bị chính bộ chuẩn hoá lỗi đọc lại thành 3,85 nên kiểm số ở S9 trượt); 2 lệch đầu ở S7.
- S5: 43 ca đỏ; 10 giữ; 33 lệch đầu ở S3 (bộ che phạm vi của vết dùng cùng `InScope` nên S3 đã hiện "‹ngoài phạm vi›" trước khi tới S5).
- S9: 15 ca đỏ; 14 giữ; 1 lệch đầu ở S7.
- S3 (32/32), S4 (70/70), S6 (25/25), S8 (3/3), S10 (10/10): không ca nào bị loại.

## Biến thể đã thử và bỏ (cùng tầng S4) [ĐO]

- đảo tie-break số cụm từ khoá (`score >` thành `<`): 2 ca đỏ, cả hai S4 (ít hơn 5);
- chọn thuật ngữ chỉ số CUỐI thay vì đầu trong `metricOf`: 0 ca đỏ;
- lấy từ khoá ở mệnh đề TRƯỚC mệnh đề trọng tâm (`in.Raw[:fs]` thay `in.Raw[fs:fe]`): 4 ca đỏ, cả bốn S4 (ít hơn 5);
- bỏ chặn nhiều kỳ trong khe period; bỏ chặn hai thực thể cùng loại: 0 ca đỏ.
Giữ lại biến thể đảo ưu tiên ý định (70 ca đỏ, đều S4).

## Điều đáng biết cho người đọc kết quả

1. [ĐO][CODE] S8 chỉ có 3 thẻ. Bảng vết ở S8 chỉ ghi "model diễn đạt" / "mẫu câu của tenant <mẫu của router>" / "model chưa được gọi" kèm outcome; bản nháp, mẫu câu thực dùng,
   `plan.Template` đổi sau S5 đều không hiện riêng ở S8 (mẫu đã nằm ở đầu ra S5; chữ câu trả lời chỉ hiện ở S10). Nên lỗi nằm ở dựng câu (render, displayNumber, chọn confirm_yes / confirm_no)
   chỉ lộ đầu tiên ở S9 hoặc S10, không phải S8. Chỗ duy nhất S8 là bước lệch đầu: nhánh thoát sớm của RAG (không có span generate, lý do chứa outcome).
   Bộ 423 ca ĐẠT chỉ có 3 ca đi nhánh đó (117 / h1 / h3, đều lớp unsupported_answer); ca thứ tư (RAG + INSUFFICIENT_EVIDENCE) đi nhánh truy xuất rỗng, nơi S6 lệch trước.
2. [ĐO][CODE] `eais trace diff` (traces.FirstDifferentStep) KHÔNG cho cùng đáp án với so từng bước: nó chỉ so Decision và quy ước verdict→S9, outcome→S6, bằng chứng→S7, câu→S8.
   Với 38 thẻ đỏ: 25 trùng tầng gieo; 13 lệch (S10 → "S8" 5 thẻ; S6 → "S8" 5 thẻ; S8 → "S6" 3 thẻ). Nhãn của phép đo này dựa trên so từng bước (đúng như lệnh), không dựa trên `trace diff`.
3. [ĐO] Thời gian trong bảng vết không phân biệt đỏ / xanh: tổng ms trung vị 1,0 ở cả hai nhóm (đỏ trung bình 1,3; xanh 1,1).
4. [ĐO] Phân bố thẻ: đỏ theo bộ h3 20, 117 8, h4 6, h1 4; theo lớp representation 10, routing 5, source_conflict 5, ambiguity 4, unsupported_answer 4, semantic 4, permission 3, entity 3.
   Xanh theo bộ h3 12, 117 9, h4 9, h1 8; theo lớp representation 7, semantic 6, routing 5, source_conflict 5, unsupported_answer 5, permission 4, ambiguity 3, entity 3.
   Xanh chọn vòng tròn theo (bộ, lớp) nên đều hơn đỏ; đỏ lệch về h3 và representation vì ứng viên của lỗi dồn ở đó. Người đọc có tiên nghiệm yếu nếu đoán theo lớp.
5. Chọn thẻ: vòng tròn theo (bộ, lớp), thứ tự theo sha256, bỏ ca có câu hỏi giống ≥ 0,78 (SequenceMatcher) với thẻ đã chọn; mỗi ca chỉ dùng một lần trong toàn bộ thẻ.
   Tên thẻ the-001... xếp theo sha256 của "<bộ>:<id ca>|<nhãn binary hoặc goc>".
