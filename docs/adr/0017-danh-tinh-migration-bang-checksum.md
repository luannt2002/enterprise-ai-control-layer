# ADR-0017 — Danh tính migration bằng checksum; cấm đổi số hay sửa nội dung migration sau khi đẩy

- **Ngày:** 24/10/2026.
- **Trạng thái:** **CHẤP NHẬN 24/10/2026** — theo GO chung 24/10 của chủ dự án cho các khuyến nghị của báo cáo audit vNext (§11.6, dòng
  ADR-0017). Mã đã thi công cùng ADR này: `platform/pgstore/migrate.go`.
- **Đính chính 24/10 (sau lát L5b):** comment của `0010_tenant_khong_rong.sql` về dữ liệu bẩn nay sai — xem mục "Đính chính" cuối tệp.
- **Giải:** F-036 (migration nhận diện bằng tên tệp). **Liên quan:** F-037 (migration `0010_tenant_khong_rong.sql` là migration đầu tiên áp
  theo luật này); báo cáo `docs/bao-cao/2026-10-24-vnext-audit/BAO-CAO.md` §01, §10, §11.6.

## Bối cảnh

Trước ADR này `Migrate` coi một tệp đã áp khi `schema_migrations` có dòng mang đúng TÊN tệp. Tên đã đổi số ba lần ngay trong merge
(`521867f`, `8c72fe2`, `71c6b45`; luật cũ "bên gộp main trước lấy số"). Hệ quả đo được trên một database dev ngày 24/10:
`0004_tai_lieu_phien_ban.sql`, `0005_…`, `0006_…` và `0007_tai_lieu_phien_ban.sql` cùng nằm trong `schema_migrations` — một migration được
áp BỐN lần dưới bốn tên; `0007_vector_tai_lieu.sql` và `0008_vector_tai_lieu.sql` cũng vậy. Lần áp lại vô hại chỉ vì các tệp viết
`IF NOT EXISTS`; một tệp không chạy lại được sẽ hỏng giữa chừng, và một tệp bị sửa nội dung sau khi áp thì không ai biết.

## Quyết định

1. **Danh tính = tên tệp + checksum nội dung.** `schema_migrations` có thêm cột `checksum` = sha256 hex của tệp (bỏ `\r` của CRLF để bản
   checkout trên Windows không thành "nội dung đổi"). Mỗi lần áp ghi cả hai.
2. **Tương thích ngược.** Cột thêm bằng `ADD COLUMN IF NOT EXISTS`; dòng ghi trước ADR này (checksum `NULL`) được điền khi gặp lại tệp cùng
   tên. Bản `Migrate` cũ vẫn chạy được trên bảng mới (nó chỉ đọc / ghi `version`).
3. **Đối chiếu trước, áp sau.** Trước khi áp bất kỳ tệp nào, `Migrate` đối chiếu toàn bộ `schema_migrations` với các tệp. Có lệch thì trả
   lỗi (nhãn `infra:postgres`) liệt kê mọi chỗ lệch và **không áp tệp nào**:
   - **(a) nội dung đổi:** tệp đã áp theo tên mà checksum khác checksum đã ghi → lỗi.
   - **(b) đổi tên / đổi số:** tệp chưa áp theo tên nhưng `schema_migrations` có dòng tên khác **cùng checksum**, hoặc **cùng phần tên sau
     số** (`0006_kho_bao_cao_ngay.sql` ↔ `0009_kho_bao_cao_ngay.sql` — cách duy nhất nhận ra dòng ghi trước khi có checksum) → lỗi kèm
     hướng dẫn, **không chạy lại**. Chọn lỗi thay vì "báo rồi tự bỏ qua": `Migrate` không có kênh báo nào ngoài lỗi, tự ghi tên mới sẽ
     giấu một lỗi gộp, và nhận theo phần tên sau số là phép đoán cần người xác nhận.
   - Dòng không ứng với tệp nào mà cũng không trùng tệp đang chờ nào (như các dòng thừa ở database dev nói trên) **không** bị báo:
     không có tệp nào bị chạy lại hay bỏ sót vì chúng; chặn ở đây là báo oan.
4. **Luật:** migration đã đẩy lên nhánh dùng chung thì **không đổi số, không đổi tên, không sửa nội dung — kể cả comment**. Sửa lược đồ
   bằng migration mới. Hai nhánh cùng lấy một số: nhánh gộp sau đổi số tệp của mình **trước khi đẩy**; tệp đã áp ở database nào thì
   database đó phải sửa dòng theo mục dưới.

## Xử lý khi `migrate` báo lệch (vai quản trị)

- **(a)** Lấy lại nội dung tệp từ git (`git log -p -- db/migrations/<tệp>`). Thay đổi là cố ý thì hoàn tác tệp và viết migration mới.
  Chỉ khi đã đọc diff và chắc chắn nó không đổi lược đồ (ví dụ chỉ sửa comment) mới được `UPDATE schema_migrations SET checksum = NULL
  WHERE version = '<tệp>'` để lần chạy sau điền lại.
- **(b)** Đối chiếu nội dung tên cũ (`git show <commit>:db/migrations/<tên cũ>`) với tệp hiện tại. Giống nhau →
  `UPDATE schema_migrations SET version = '<tên mới>' WHERE version = '<tên cũ>'`. Khác nhau → xoá dòng tên cũ để tệp mới được áp lại
  (chỉ khi tệp mới chạy lại được an toàn), hoặc dựng lại database.

## Hệ quả

- Lần chạy đầu của bản mới trên database đang dùng chỉ thêm cột, điền checksum và áp tệp mới; lần sau không làm gì (đo 24/10: database
  dựng bằng bản `3f1bf7e` có 9 dòng, không cột checksum → `eais migrate` bản mới: 10 dòng, 10 checksum → chạy lại: không đổi, rc=0; bản
  cũ chạy trên bảng có checksum: rc=0, không đổi).
- Sửa comment một migration cũ giờ làm mọi database báo (a) — đúng ý mục 4.
- **Giới hạn đã biết:** dòng cũ được điền bằng nội dung tệp HIỆN TẠI, nên một tệp đã bị sửa TRƯỚC khi có ADR này không bị phát hiện.
  Hai tệp khác số nhưng cùng phần tên sau số cũng bị coi là đổi số — đặt tên migration mới cho khác.
- Test: `TestMigrateDanhTinhBangChecksum` (bảng kiểu cũ, (a), (b) có / không checksum, đổi hẳn tên, dòng thừa không báo oan),
  `TestChecksumMigrationBoQuaCRLF`.

## Phương án đã cân nhắc

- **Chỉ đặt quy ước "không đổi số", không máy ép:** bị loại — quy ước đã vỡ ba lần trong một tuần.
- **(b) tự ghi tên mới rồi bỏ qua:** bị loại — giấu lỗi gộp, và phép nhận theo phần tên sau số có thể sai.
- **Lỗi với mọi dòng không ứng với tệp nào:** bị loại — database dev thật có bốn dòng như vậy, vô hại; lỗi sẽ chặn oan.
- **Dùng công cụ migration bên ngoài:** bị loại — thêm phụ thuộc cho một thay đổi +71 / −11 dòng ở `migrate.go`.

## Đính chính 24/10/2026 — 0010 gặp dữ liệu bẩn thì app KHÔNG khởi động

Comment đầu `db/migrations/0010_tenant_khong_rong.sql` (dòng 9–10) ghi: database đã có dòng tenant `''` thì cả tệp hoàn tác, "ứng dụng vẫn
chạy vì Ready không đòi 0010". Câu đó đúng lúc viết, nay **sai**. Tệp migration không sửa được, kể cả comment (mục 4), nên đính chính ở đây.

Chuỗi đọc mã (`f589cf6`):
1. `Migrate` áp các tệp chờ theo thứ tự tên; tệp nào lỗi thì trả lỗi ngay, tệp sau **không** áp (`platform/pgstore/migrate.go`, vòng `pending`).
2. Lát L5b thêm `0011_ghi_chu_suy_luan.sql` (cột `facts.note`) và `Ready` đòi cột đó (`platform/pgstore/ready.go:44`,
   `readyColumns`; commit `f589cf6`).
3. Vậy database có dòng tenant `''` ⇒ 0010 hỏng ⇒ 0011 không áp ⇒ `Ready` báo "cột facts.note chưa có (migration 0011)" ⇒ `eais serve`
   không khởi động. Đóng mặc định — đúng hướng (không phục vụ khi lược đồ chưa đủ), nhưng người vận hành phải biết nguyên nhân thật nằm ở 0010.

Chưa có test chạy trọn chuỗi này trên PG (dòng tenant `''` → `eais migrate` → `eais serve`); ba mắt xích trên là đọc mã.

**Xử lý (vai quản trị):**
1. Đọc lỗi của `eais migrate`, không phải của `serve`: lỗi nêu `migration 0010_tenant_khong_rong.sql` và ràng buộc
   `<bảng>_tenant_khong_rong` bị vi phạm. Thông báo "chạy `eais migrate`" của `Ready` chỉ là hệ quả.
2. Đếm dòng tenant `''` từng bảng bằng câu `SELECT` ghi ở comment đầu tệp 0010.
3. Với từng dòng: thuộc một tenant thật thì `UPDATE … SET tenant_id = '<tenant>'`; là rác thì chép ra trước (`COPY (SELECT …) TO …`) rồi
   `DELETE`. `decision_records`, `usage_records` là vết (chỉ thêm, 0003): chép ra trước khi xoá, không bao giờ xoá mà không giữ bản.
4. Chạy lại `eais migrate` (0010 rồi 0011 áp), rồi `eais serve`.

Không có đường tắt: không sửa tệp 0010 (mục 4), không ghi tay dòng 0010 / 0011 vào `schema_migrations` (lược đồ sẽ thiếu ràng buộc / cột
mà sổ khai là đã áp).
