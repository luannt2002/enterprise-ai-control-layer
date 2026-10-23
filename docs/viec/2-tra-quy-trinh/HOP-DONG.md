# Hợp đồng việc 2 — tra căn cứ đang hiệu lực (RAG)

> **Trạng thái:** ĐỀ XUẤT; mô tả đúng mã trên nhánh `rag/0-viec` (chưa gộp `main`). Cổng QuânIT: `CONG-QUANIT.md`. Kế hoạch: `KE-HOACH.md`. Roadmap: `docs/ROADMAP-RAG.md`.
> **Dữ liệu:** giả lập; mọi kết luận "trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt" (D-RM-03).

## 0. Tóm tắt

- **Vào:** danh tính đã xác thực (tenant, vai, `entity_scope`), câu hỏi, k, ngày hỏi (đồng hồ của lượt).
- **Ra:** đoạn trích của bản đã duyệt, đang hiệu lực, người hỏi được đọc, kèm `mã@vN#dòng`; hoặc một **thông báo phiên bản** (xung đột / không có bản hiệu lực); hoặc rỗng.
- **Luật chính:** BR-DOC-01 (chỉ bản hiệu lực; hai bản cùng hiệu lực = xung đột), BR-DOC-02 (trích dẫn có mã, bản, vị trí), BR-DOC-03 (bản không qua cổng chất lượng đọc thì không công bố) — `docs/tenants/construction/01-business-map.md:415-417`.

## 1. Làm và không làm

| Làm | Không làm |
|---|---|
| Truy hồi đoạn tài liệu có phiên bản, quyền, phạm vi | Số vận hành (kho xuất, máy đốt…): đi SQL / CALC |
| Báo xung đột / không có bản hiệu lực | Chọn ngầm một bản khi mâu thuẫn |
| Cắt bảng: chỉ khi dòng sau là dòng phân cách; mỗi hàng một đoạn kèm header, chỉ chữ của hàng được chấm điểm; bảng viết liền mang header riêng | Dùng định mức trích từ bảng làm đầu vào CALC (U4b, cần phối hợp) |
| Chạy ca thật ở mức truy hồi (`evals/rag`) | Diễn đạt câu trả lời, kiểm S9, vết (U7, nối pipeline) |

## 2. Mô hình tài liệu

Tệp markdown, phần đầu tuỳ chọn giữa hai dòng `---`:

```yaml
id: SOP-NL-ĐM-01         # mã logic; các bản cùng mã dùng chung ACL trong acl.json
version: 3               # số nguyên ≥ 1
status: approved         # approved | draft | needs_review | superseded
effective_from: "2026-03-01"   # ngày, khoảng [từ, đến); bỏ trống = không giới hạn
effective_to: ""
scope: [PRJ-A]           # khoá thực thể được đọc; bỏ trống = chỉ xét vai
removed_columns: []      # cột bảng bỏ CÓ CHỦ ĐÍCH so với bản trước (cổng chất lượng không coi là mất cột)
```

- Tệp không có phần đầu: nạp như cũ (id = tên tệp, không phiên bản, coi như đã duyệt, luôn hiệu lực).
- **Báo lỗi lúc nạp** (không lặng lẽ công bố): khoá lạ; trạng thái lạ; ngày sai dạng; `version` thiếu hoặc không nguyên; đến ≤ từ; thiếu dòng đóng; trùng `id@bản`; cùng id vừa có tệp không phiên bản vừa có tệp khai phiên bản. Tệp CRLF hay có BOM vẫn đọc đúng phần đầu.
- Mã đoạn: `mã#dòng` (tài liệu không phiên bản, giữ dạng cũ) hoặc `mã@vN#dòng`; `dòng` = vị trí dòng trong thân tài liệu.

## 3. Vòng đời một bản

```text
nháp (draft) ──duyệt──► đã duyệt (approved) ──có bản mới thay──► đã thay thế (superseded)
      │                        │
      └─ cổng chất lượng đọc trượt (BR-DOC-03) ─► chờ soát (needs_review): không công bố
```

- Hiệu lực xét theo **ngày hỏi**, không theo ngày nạp.
- Tài liệu mà mọi bản đều "đã thay thế" thì im lặng như không tồn tại (đã được thay bằng tài liệu khác).
- Cổng chất lượng đọc tự động lúc nạp (U5a, BR-DOC-03 "bảng đủ ô, số đủ"; soát 2 vòng). Bản khai "đã duyệt" bị chuyển "chờ soát" kèm lý do (`Hold`, gọn; hiện trong thông báo phiên bản) khi:
  - một hàng thiếu / thừa ô so với header, một ô rỗng, hoặc một bảng chỉ còn header. **Ô trống có chủ đích phải ghi `-`**;
  - mất cột so với **mốc**: số lần mỗi cột xuất hiện, đếm **một lần mỗi bảng** (không theo số hàng), giảm — trừ cột khai trong `removed_columns`;
  - `removed_columns` khai một cột không có ở mốc (khai sai tên, hoặc chép sót từ bản trước).
- **Mốc** = bản đã công bố gần nhất (đã duyệt hoặc đã thay thế) cùng mã, **bảng của nó đạt kiểm**, và **phạm vi của nó bao trùm phạm vi bản mới** (không phạm vi = mọi người). Người đọc được bản mới thì đọc được mốc, nên lý do giữ không lộ gì. Bản đã thay thế không đạt kiểm thì không bị giữ (đã không còn phục vụ) nhưng không làm mốc.
- Dòng phân cách: ngay sau header, theo luật GFM (`| --- |`, `| :-: |`, `|-|`) và **cùng số ô** với header. Giữa thân bảng, chỉ dòng mà mọi ô có ≥ 3 gạch mới mở bảng viết liền; `| - | - |` là hàng dữ liệu ("không áp dụng"). Đây là thói quen viết, không phải luật GFM.
- Giới hạn đã biết:
  - cổng chạy lại mỗi lần nạp, không nhớ bản nào đã công bố: nhập một bản cũ hơn, sửa một bản cũ, hay làm luật cổng chặt hơn đều có thể hạ một bản đang công bố (soát U5a, G5). U6 phải lưu trạng thái công bố — U6a (23/10) CHƯA làm phần này, còn nợ ở U6c (KE-HOACH).
  - nới rộng phạm vi (ví dụ [PRJ-A] → mọi người) mà không có mốc bao trùm thì không so được cột.

## 4. Truy hồi (`intelligence/rag`, qua `KnowledgeRetriever` ở `core/pipeline/ask.go:43`)

Hình dạng contract **không đổi**: `Retrieve(ctx, Principal, query, k) ([]contracts.Evidence, error)`.

1. Tenant của người hỏi phải là tenant của chỉ mục; sai → lỗi chính sách.
2. Tài liệu **thấy được** = vai có trong ACL **và** (không khai phạm vi **hoặc** phạm vi giao với `entity_scope`; bên gọi có thể đưa `InScope` của policy). Không thấy = không tồn tại: không đoạn, không thông báo.
3. Bản **phục vụ được** = trạng thái đã duyệt (hoặc không khai) **và** hiệu lực vào ngày hỏi.
4. Chấm điểm từ khoá (IDF, luật dấu, bỏ dấu); mỗi tài liệu chỉ góp đoạn của đúng một bản phục vụ được.
5. Nếu tài liệu khớp nhất **không có đúng một** bản phục vụ được và điểm của nó ≥ đoạn tốt nhất còn lại → trả `*rag.VersionNotice` (≥ 2 bản: `version_conflict`; 0 bản: `no_effective_version`) kèm mọi bản người hỏi thấy. Hoà điểm giữa hai tài liệu có vấn đề: chọn theo mã.
6. Phá hoà điểm giữa các đoạn: tài liệu → vị trí dòng. Thứ tự cộng điểm cố định.

**Pipeline xử lý thông báo (U7a, đã làm):** `*contracts.KnowledgeNotice` (bí danh `rag.VersionNotice`) không phải lỗi hệ thống. `answerKnowledge` đổi nó thành mẫu câu: xung đột → `knowledge_version_conflict`, outcome `SOURCE_CONFLICT`; không có bản hiệu lực → `knowledge_no_effective`, outcome `INSUFFICIENT_EVIDENCE`; 0 lời gọi model; câu HYBRID nối mẩu `hybrid_*` sau phần số liệu đã kiểm. Truy xuất hoặc model lỗi trong câu HYBRID → giữ phần số + `hybrid_knowledge_error`; câu chỉ hỏi quy trình thì vẫn là lỗi lượt. Dữ liệu không phiên bản không bao giờ sinh thông báo.

## 5. Ca → cổng

| Ca | Nấc | Kỳ vọng ở mức truy hồi | Hiện trạng [ĐO `go test -v ./evals/rag/`] | Còn thiếu |
|---|---|---|---|---|
| CON-T-005 | RAG-1/2 | đoạn v3, hàng máy xúc kèm header "L/giờ", không số của v2 | xanh | diễn đạt + trích dẫn (U7) |
| CON-T-032 | RAG-1 | v3 khai "approved" nhưng mất cột → cổng giữ lại; không có bản hiệu lực, nêu v2 + v3 | xanh | báo chủ tài liệu (U7) |
| CON-T-061 | RAG-1 | xung đột, nêu v2 + v3 | xanh | route chuyển người (U7); cache theo bản (U9) |
| CON-T-071 | RAG-3 | tenant khác: rỗng, dù ACL có vai của người gọi | xanh | mẫu câu fallback (U7) |
| CON-T-004 | RAG-1 | đoạn hàng máy xúc 1,2 m³, không lọt máy ủi / xe lu | **đỏ biết trước** | EX-01 → loại máy chưa có trong dữ liệu mẫu; định mức vào CALC (U4b) |
| CON-T-006, 008, 094 | RAG-2 | quy trình nhiều bước / điều khoản hợp đồng | chưa có dữ liệu mẫu | U3b (đường dẫn mục) |
| CON-T-081, 082 | RAG-1 / RAG-4 | bộ đọc sập; model hết giờ | chưa chạy | U5, U7 |

## 6. Cổng ra theo nấc

| Nấc | Cổng ra | Trạng thái |
|---|---|---|
| RAG-0 | giấy này + `CONG-QUANIT.md`; ca chạy thật; baseline (nạp thẳng, RAG làm trung tâm) | giấy + ca: có; baseline ở mức bằng chứng: có (U8a — EAIS 5/5, hai đối chứng 1/5); baseline ở mức câu trả lời (cần model): chưa |
| RAG-1 | nạp sai khai báo → lỗi; 0 bản hết hiệu lực / chưa duyệt làm căn cứ; xoá → không còn thấy | phần chỉ mục trong bộ nhớ: đạt [ĐO]; xoá + Postgres: U6 |
| RAG-2 | 0 bảng mất header; 0 dòng phân cách thành đoạn; phủ hết văn bản | đạt cho bảng [ĐO]; đường dẫn mục: U3b |
| RAG-3 | 0 rò (tenant, vai, phạm vi, từng bản); Recall@k trên bộ phát triển | 0 rò: đạt [ĐO]; Recall@k: U8 |
| RAG-4 | 4 trạng thái hiện đúng cho người hỏi; trích dẫn trỏ đúng 100% | U7 |
| RAG-5 | bộ giữ lại chạy một lần; thắng baseline | U8 |

## 7. Điểm tích hợp (vùng dùng chung — phối hợp, không tự sửa trước khi báo team)

| Đơn vị | Tệp | Thay đổi |
|---|---|---|
| U4b | `tenants/construction/seed/entities.yaml`, pack, CALC trong `core/pipeline` | thuộc tính loại máy cho thiết bị; định mức từ bản hiệu lực làm đầu vào CALC (BR-FUEL-05) — chủ dự án chốt nguồn có thẩm quyền |
| U5 | `data/ingestion/readers.go` (chỉ gọi, không sửa) | bộ nạp RAG gọi bộ đọc sẵn có; cổng chất lượng đọc |
| U6 | `platform/pgstore`, `db/migrations` | bảng tài liệu có bản / trạng thái / hiệu lực / phạm vi + RLS + trạng thái công bố. **U6a xong 23/10:** migration 0005 (cột + khoá `(tenant_id, id, version)` + CHECK ngày), `pgstore` ghi / đọc đủ trường, `Ready` từ chối kho chưa áp 0005; 5 ca bộ nhớ = PG. Chặn cũ (soát U5a, G11: PG phục vụ bản nháp / bị giữ / có phạm vi như bản chính, trùng mã vấp khoá chính) đã gỡ. Còn: trạng thái công bố (U6c), `tsvector` (U6b). Vận hành: chỉ nạp tài liệu có phiên bản khi mọi binary trước U6a đã dừng — binary cũ đọc bảng mới mà bỏ qua cột trạng thái / hiệu lực / phạm vi |
| U7 | `core/pipeline/ask.go`, `intelligence/models`, pack `policy.yaml` | `VersionNotice` → mẫu câu + outcome; HYBRID giữ phần số; escape thẻ khung; `Now` của lượt và `InScope = policy.InScope` (xong ở U6a: pipeline truyền qua ngữ cảnh lượt); vai `site_supervisor` / `equipment_manager` |
| U9 | `observability/` | vết RAG, nhãn lỗi |

## 8. Nợ và chưa đo

- ~~`Now` / `InScope` của lượt~~ — trả ở U6a: pipeline đặt `contracts.KnowledgeQuery{AsOf, InScope: policy.InScope}`, thắng trường của chỉ mục; phát lại dùng ngày đã ghi. Còn `shortcut:` đồng hồ máy chỉ cho bên gọi ngoài pipeline.
- Tài liệu có phạm vi: ACL và phạm vi phải thoả bởi CÙNG một vai (soát độc lập U6a: accountant scope "all" không cho site_engineer mượn). Đường số liệu (`router.Policy.InScope`) vẫn gộp phạm vi của mọi vai — ngoài phạm vi nhánh RAG, ghi để chủ dự án xét.
- ~~IDF tính trên mọi tài liệu của tenant~~ — sửa 23/10: IDF chỉ trên đoạn người hỏi thấy, đường bộ nhớ = đường PG từng câu (S7D-25 là ca đo ra lệch, `S7-BO-CA-DOC-LAP.md` §2).
- **P1 — biết im:** sửa 23/10 bằng luật 2' (chủ đề = câu bỏ số / ngày / mã của S2; im khi ≥ 1/2 từ của chủ đề ngoài kho người hỏi thấy): câu `khong_co` của bộ S7 phát triển nhận đoạn 5/5 → 2/5, top-3 giữ 30/36. Còn 2/5 — `S7-BO-CA-DOC-LAP.md` §2.
- Bảng: mỗi hàng một đoạn; bảng rất dài / rất rộng cần gom nhóm theo ngân sách token khi có tài liệu thật (`shortcut:` trong mã).
- Đoạn hàng bảng có 2 dòng (header + hàng). Trước khi đưa tài liệu có bảng vào kho của pipeline (U7), cần ca kiểm: model trích nguyên văn sẽ in cả dòng header; prompt Messages ghi `[mã] header` rồi dòng hàng không có mã; bộ chặn lượng từ tài liệu thấy đơn vị ở dòng trên số (soát U3, T6).
- Chưa đo trên tài liệu thật của doanh nghiệp nào.
