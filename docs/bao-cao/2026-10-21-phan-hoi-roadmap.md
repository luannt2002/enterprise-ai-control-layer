# Phản hồi của team về `docs/ROADMAP.md` — BẢN GIỮA CHỪNG (21/10/2026)

Trả lời `docs/PROMPT-AUDIT-ROADMAP.md`, theo khuôn §5 của prompt.

Phạm vi lượt này do chủ dự án chốt (D-OP-13, `docs/CHO-QUYET-DINH.md` mục 0):
- **Làm hôm nay:** chỉ phần đọc mã K6–K15.
- **Để ca sau:** K1–K5 (bơm dữ liệu), Q1–Q13, bàn §4.7 và Phụ lục C.
- Q1 và Q8 trả lời sớm vì đã có số đo trong ca.

Nhãn bằng chứng giống ROADMAP: [ĐO] [CODE] [REPO] [SUY].

## 1. Mốc

- **HEAD lúc bắt đầu:** `e62edeb`. Mã Go không đổi từ `8280f40`: `git diff --stat 8280f40 HEAD -- '*.go'` rỗng [ĐO].
- **HEAD lúc push:** xem commit chứa tệp này.
- **Môi trường:** Go 1.26.4; PostgreSQL dev (`eacl-pg`) có chạy. Người kiểm K6–K15 không dùng PG.

## 2. Đo lại Phụ lục C

CHƯA LÀM, để ca sau theo phạm vi D-OP-13.

## 3. K1–K15

**Cách kiểm:**
- Một người kiểm độc lập (agent chỉ đọc) đọc mã tại `ac98cf4` và chạy test có sẵn. Người này không viết các khẳng định, không viết phần mã liên quan, không sinh hay bơm dữ liệu.
- Phiên BUILD đo lại bằng chứng ở K6, K7, K8, K9, K12, K13, K14, K15 và ở hai lỗi ROADMAP (đọc đúng dòng, chạy lại 9 test có sẵn).
- Mã Go không đổi từ `8280f40`, nên số dòng của bảng gợi ý vẫn đúng.

| Mã | Kết quả | Bằng chứng | Ghi chú |
|---|---|---|---|
| K1–K5 | CHƯA ĐO | — | Cần bơm dữ liệu; ca sau (D-OP-13) |
| K6 | XÁC NHẬN | **Harness chỉ nạp tenant đang chạy:**<br>- `winrun.go:116-118` (`dir = Join(cfg.Data, cfg.Tenant)` → `Ingest`), `:158-173` (danh bạ, tài liệu, chỉ mục, PG đều theo `cfg.Tenant`);<br>- `g7run/main.go:633` cùng mẫu.<br>**Chuỗi tenant B:** chỉ nằm ở trường `not_contains` / `note` của H1 (198) và H3 (6 + 5), 0 ở câu hỏi [ĐO, đếm theo tên trường YAML] | "canary 0" đúng theo cấu trúc: B không bao giờ được nạp, câu hỏi không nhắc B. Đó không phải bằng chứng cách ly.<br>**Đối chứng dương đã có:**<br>- cấp bộ chấm / baseline: `TestG7CounterCatchesWhatRunWinMisses` (`evals/g7_test.go:543`), `TestBangCongCanary` (`kiemcongcu_test.go:177`), `TestNameCanaries` (`g7run_test.go:363`), `TestOnlyOneTenantData` (`baseline_test.go:158`);<br>- cấp thành phần: `TestRLSCanaryB`, `rag_test.go`.<br>**Cấp harness: CHƯA có** ⇒ việc §4 mục 3 của brief BUILD |
| K7 | XÁC NHẬN | **Mã:**<br>- `pgstore.go:24-34` chỉ `pgxpool.New` + `Ping`;<br>- `load.go:27-28` `DELETE FROM …` không `WHERE`, dựa vào RLS;<br>- grep `rolbypassrls`, `rolsuper`, `pg_roles` trong mã: chỉ có ở `db/migrations/0001_win.sql` (tạo vai `eacl_app NOSUPERUSER NOBYPASSRLS`).<br>**Tài liệu:** `docs/01-architecture.md:378` hứa "khởi động từ chối" | Tài liệu hứa nhưng mã chưa làm. Nếu DSN trỏ nhầm vai quản trị, `Load` xoá dữ liệu MỌI tenant [SUY, chưa chạy] ⇒ việc §4 mục 3 |
| K8 | XÁC NHẬN | **`ask.go`:**<br>- `:470-471` đặt sẵn `Pass: true`, `accept`, `RESOLVED`;<br>- nhánh hybrid của `fail` (`:482-487`) chỉ nối mẩu `hybrid_no_knowledge` và thêm `Violations`.<br>**`templates.yaml:35`:** "Chưa có tài liệu bạn được xem…" | Cùng một mẩu câu dùng cho hai nguyên nhân: "không có đoạn nào được xem" (đúng lý do) và "S9 chặn bản nháp" (sai lý do). Test hiện có chỉ khẳng định `Violations`, không khẳng định `Pass` hay câu chữ. Bỏ phần tài liệu là thiết kế có chủ ý; lỗi nằm ở lý do được nêu |
| K9 | XÁC NHẬN (đường Ask) | `Draft.Usage` chỉ được tạo ở `messages.go:189`; đường Ask không đọc nó; không có bảng `interactions` (migration chỉ có 4 bảng) | `g7run` tự đo chi phí qua hook HTTP (`evidence.go:131-161`) và ghi Decision vào bằng chứng thô khi có `-out`. Nợ lưu bền đã ghi ở STATUS |
| K10 | XÁC NHẬN | `ask.go:463-468` (đường số liệu) và `:488-492` / `:532-534` (đường tài liệu) là hai chỗ duy nhất sinh `SYSTEM_FAILURE` + `fallback_unverified` | Ngoại lệ là phần tài liệu của HYBRID (K8) |
| K11 | XÁC NHẬN phần H3 · ĐÃ ĐỔI phần H4 | H3 chạy một lần lúc 13:08, công bố ở `9091c84`, nay là bộ hồi quy (ADR-0006 §10). D-OP-13 (`ac98cf4`): lượt chính W-G7 dùng **H5**; H4 chấm vòng sửa S4 | `HOP-DONG-G7.md` §12 (`:200-201`) và dòng `:33` nay lệch D-OP-13. Cột Trả lời của D-H1-3 còn trống, dù D-OP-13 đã trả lời câu đó (mục 8) |
| K12 | XÁC NHẬN | **625:** `OUT_OF_SCOPE` từ nhánh "không ý định nào khớp" (`router.go:185-190`); câu trả lời không có số.<br>**632:** `INSUFFICIENT_EVIDENCE` vì S4 đọc chữ "lệch" của vế quy trình thành chỉ số chênh lệch (cụm C5) | Không ca nào rò dữ liệu. Cả hai là lỗi S4: một ca không nhận ra ý định, một ca đọc sai ý định |
| K13 | MỘT PHẦN | **ĐÚNG:**<br>- `docs/v3/06` ghi R0-02, R0-06 là BLOCKED dù D-V3 đã trả lời. R0-06 có thể còn chặn thật vì cần bản `05` "đã duyệt";<br>- `docs/v3/01` đo ở `85108c4`;<br>- `docs/v3/05:73` còn "temperature 0", trái D-G7-1.<br>**ĐÃ ĐỔI:** D-OP-9 đã vào dòng "còn mở" ở `7998751`; D-G7-18, D-H1-1/2/3 vẫn thiếu.<br>**Đúng một phần:** chuỗi "G1–G7" chỉ có ở `README.md:14`; STATUS dùng "G1–G5" và bảng G1…G7 | — |
| K14 | MỘT PHẦN | `pipeline_xac_nhan_an_toan_test.go:58-68`: chú thích nói "phiếu 912", "ca 2", "phiếu 12-45" vẫn được phán quyết, nhưng 4/6 dòng kỳ vọng nay là rỗng (đổi ở `aa0e8d4`; tên và chú thích không đổi). 2/6 dòng vẫn đòi "Đúng." / "Không." | Cùng lỗi chú thích ở test anh em `TestCauXacNhanSoNoiSauLuongDungChungDonVi` (`:74-82`: chú thích viết "nên vẫn phán quyết", cả 2 kỳ vọng rỗng). ROADMAP chưa nêu chỗ này |
| K15 | BÁC | `CLAUDE.md:647-650` đúng chữ, nhưng không ngược với V3: chính V3 giao context builder "đọc có lọc quyền" (`docs/v3/06:310` R3-04; `docs/v3/03:47` V3-SEC-02) | `CLAUDE.md` §9 đặt quyền SỞ HỮU ở phần mềm doanh nghiệp; §13 nói về TRÁCH NHIỆM xử lý, không nói sở hữu. Không sửa `CLAUDE.md` (D-OP-13) |

**Test có sẵn đã chạy lại:** `env -u EACL_PG_ADMIN_DSN -u EACL_PG_DSN go test -count=1 -run '<9 tên>' ./tenants/construction/ ./evals/ ./evals/baseline/ ./intelligence/rag/ ./platform/pgstore/`. Kết quả:
- ĐẠT (8): `TestHoiGhepSoLieuVaQuyTrinh`, `TestMauCauChenSoLaBiChan`, `TestOutcomeCoKieu`, `TestClaimKhongGanBangChungBiChan`, `TestCauXacNhanMotVeVanPhanQuyet`, `TestG7CounterCatchesWhatRunWinMisses`, `TestOnlyOneTenantData`, `TestRetrieveTenantVaACL`;
- SKIP: `TestRLSCanaryB` (lượt này không bật PG).

### Chỗ ROADMAP sai sự thật (để phiên học sửa; BUILD không sửa ROADMAP lượt này)

1. `ROADMAP.md:1300` (cùng `:173`, `:1235`) viết rằng D-H1-3 "đã làm theo khuyến nghị". Sai:
   - khuyến nghị A gồm "lượt chính G7 = H3" (`CHO:84`); việc đó chưa làm, ROADMAP đã tự rút (§4.7 #1, Phụ lục D), và nay D-OP-13 đã thay bằng H5;
   - chỉ nửa đầu của A đã làm: H3 được đóng băng bằng băm trước khi sửa mã.
2. `ROADMAP.md:1734` viết "đối chứng dương duy nhất là `TestRLSCanaryB` và `rag_test.go`". Chữ "duy nhất" sai: còn 4 test cấp bộ chấm / baseline (xem K6). Kết luận "chưa có đối chứng dương cấp harness" vẫn đúng.
3. `ROADMAP.md:1897` khái quát K14 quá tay: 2/6 dòng của test vẫn đòi phán quyết.
4. **Lỗi thời** (đúng ở `8280f40`, nay đã đổi):
   - `:1235` và `:1892` viết "`CHO:28` thiếu D-OP-9": đã thêm ở `7998751`;
   - `:120`, `:1246`, `:1285`, `:1310`, `:2182` coi H4 là tập sạch cho lượt chính: D-OP-13 đổi thành H5.

## 4. Q1–Q13

- **Q1 (H4)** [ĐO]:
  - Mục đích: H4 chấm vòng sửa tầng ý định S4; lượt chính W-G7 dùng H5 (D-OP-13).
  - Người soạn: 2 agent soạn độc lập, chỉ đọc bộ đồ nghề (tài liệu nghiệp vụ + dữ liệu hạt giống 42), không đọc mã. Phiên viết mã chỉ ghi mù nội dung ra đĩa và chỉ thấy số đếm.
  - Đã soạn 138 ca: mỗi lớp 16–19 ca, 0 lỗi định dạng (`kiemca`).
  - Soát số liệu: hai người soát R1 báo đã tính lại 89/89 đáp án số, 0 lệch. Đây là số đếm của người soát; phiên viết mã không xem được nội dung để đo lại.
  - Kiểm toán nhiễm hai người soạn: 0 đường dẫn ra ngoài phạm vi.
  - Tiếp theo: soát hợp đồng / mơ hồ / trùng → sửa → ghép → kiểm toán nhiễm → cam kết băm (commit manifest, nội dung ở máy).
  - Chỉ chạy MỘT lần sau khi mã S4 đã đóng băng.
  - Chủ dự án không cần quyết thêm gì cho H4. Riêng H5 (cho W-G7) theo điều kiện 2–3 của D-OP-13.
- **Q8 (T2.2)** [REPO]:
  - CHƯA có người soát thứ ba cho 3 dòng vá T2.2. Các dòng vá này do người soát vòng 2 đề xuất; phiên chính áp vào và đo lại.
  - Đề xuất: một người soát độc lập (chỉ đọc) ở ca sau, TRƯỚC khi chạy H4. Vòng S4 sẽ chạy lại danh mục đột biến n43 vì cổng phán quyết đứng sau ý định.
- Q2–Q7, Q9–Q13: ca sau.

## 5. Bàn quyết định §4.7

| # | Loại | Kết quả | Commit |
|---|---|---|---|
| 1 | 2 | Chủ dự án đã quyết: D-OP-13 (B kèm 4 điều kiện) | `ac98cf4` |
| 2–16 | — | Ca sau | — |

## 6. Việc đã làm trong lượt này

- **Commit:**
  - `ac98cf4`: D-OP-13 ghi vào `CHO` mục 0;
  - commit chứa tệp này;
  - commit liền sau: nhận handoff 01, sửa giờ ghi D-OP-13, và tệp tiến độ `docs/bao-cao/TIEN-DO-BUILD.md`.
- **Không đổi mã;** không sửa ROADMAP, không sửa `CLAUDE.md`.
- **Test trước khi push:** `go vet ./...`, `gofmt -l .`, `go test -count=1 ./...` có PG. Kết quả ghi ở tệp tiến độ.
- **Soát độc lập:** một agent chỉ đọc kiểm K6–K15. Phiên BUILD đo lại mẫu bằng chứng (mục 3) và kiểm quy trình của người kiểm:
  - `go.mod` / `go.sum` không đổi, dù có một lệnh test dùng nhầm `-mod=mod`;
  - 0 id ca hay chuỗi thư mục của bộ giữ lại chưa chạy (H4) xuất hiện trong lệnh, kết quả công cụ hay câu trả lời của người kiểm.

## 7. Việc chưa làm và vì sao; rủi ro còn lại

- **Chưa làm (để ca sau, theo phạm vi D-OP-13):** Phụ lục C, K1–K5, Q2–Q7, Q9–Q13, bàn §4.7 #2–#16.
- **Không sửa `docs/ROADMAP.md` lượt này,** để tránh ghi đè lẫn nhau với phiên học. Phiên học đã nói sẽ tự sửa:
  - mâu thuẫn §4.7 #1 với việc 6;
  - dòng 1465.

  Chỗ sai sự thật mà team tìm thấy được liệt kê ở mục 3 để phiên học sửa.
- **`docs/study/` chưa ai soát nội dung.** Đã quét bí mật: 0 khoá, email, đường dẫn máy, tên cũ.

## 8. Câu cần chủ dự án trả lời

1. **D-H1-3 còn trống cột Trả lời,** dù D-OP-13 đã trả lời đúng câu đó: lượt chính chạy trên tập nào → H5.
   - Khuyến nghị: anh/chị ghi "xem D-OP-13" vào cột đó (chỉ chủ dự án ghi cột này).
   - Hệ quả nếu chưa ghi: hai câu trong `CHO` nói khác nhau về cùng một việc.
2. **`HOP-DONG-G7.md` §12 (`:200-201`) và dòng `:33` vẫn ghi tập lượt chính là H1 / H3,** lệch D-OP-13.
   - Khuyến nghị: cho phép BUILD sửa hai chỗ này ở ca sau, chỉ để chép lại D-OP-13. Hợp đồng W-G7 chưa đóng băng nên sửa được.
   - Hệ quả nếu chưa sửa: hợp đồng và quyết định lệch nhau. Không ảnh hưởng gì trước khi chạy W-G7.
