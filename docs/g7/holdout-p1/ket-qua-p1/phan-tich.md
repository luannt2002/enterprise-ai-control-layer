# Bộ giữ lại P1 — lần chạy duy nhất (23/10/2026 14:29), phân tích

Trên dữ liệu giả lập, chưa có người nghiệp vụ duyệt (D-RM-03).

## Chạy gì, ở đâu

- Bộ ca: `p1.yaml` (157 ca, 23 lớp) + thế giới bổ sung `world-p1.yaml`, đóng băng bằng cam kết băm ở `1ddd10b` (22/10), công bố nguyên văn
  ngay trước lần chạy ở `18fb206` (băm khớp `holdout-manifest.json`). Người soạn không đọc mã.
- Mã: main `18fb206` = K1 + K2 + K3 + K4 + bảy lỗi P2 C1–C7 + C8–C11 (cổng §6 của cây `b22e1e6`: 1942 test có PG, 0 mất).
- Lệnh: bộ chạy chung K4, `go test -tags giulai -count=1 -run '^TestBoChayGiuLai$' -v ./execution/commands/`, hai nền (bộ nhớ và PostgreSQL
  17, database riêng `giulai_p1`, đã xoá), không overlay. Chi tiết, băm và mã thoát: `chay.txt`. Bảng cổng nguyên văn của bộ chạy:
  `bang-cong.txt`. Kết quả từng ca và từng khẳng định: `bao-cao-bo_nho.json`, `bao-cao-pg.json`, `log.txt`.
- Chạy đúng MỘT lần, không chạy lại sau khi sửa (ADR-0006 §7, như H1–H4).

## Kết quả

**153 / 157 ca đạt, 4 ca đỏ — hai nền cho kết quả giống hệt từng ca.**

| Cổng (HOP-DONG §11.1) | Trạng thái | Lớp (đạt / ca) |
|---|---|---|
| X1 gửi đôi | **ĐỎ** | A01 7/7, A02 4/4, A02b 5/5, A03 7/7, **A04 5/6**, A04b 5/5, A05 7/7, A06 7/7 |
| X2 thiếu quyền | ĐẠT phần khuôn §12.1 khai được (ngoài khuôn: A09, A11, A12) | A07 7/7, A08 9/9, A10 7/7 |
| X3 timeout sau khi ghi | ĐẠT | A21 9/9, A22 3/3 |
| X4 nguyên tử và hỏng giữa chừng | ĐẠT | A19 4/4, A20 4/4, A23 5/5 |
| X5 dữ liệu và luật | **ĐỎ** | A13 26/26, A14 7/7, A15 10/10, A16 7/7, **A17 1/4**, A18 4/4 |
| X6 biên | không có ca (A24–A26 ngoài khuôn §12.1) | — |
| X7 không đổi đường hỏi | không có ca (A27 so hai commit, ngoài bộ chạy) | — |
| X8 đột biến | ngoài bộ chạy (danh mục `scripts/dot-bien/m4-phieu-chenh-lech.sh`; ngoại lệ M5 chờ Q-P1-14) | — |
| X9 cửa vào và vết | ĐẠT phần khuôn khai được (ngoài khuôn: A28) | A29 3/3 |

## Bốn ca đỏ — một nguyên nhân

| Ca | Lớp | Khẳng định đỏ | Mọi khẳng định khác |
|---|---|---|---|
| CON-T-728 | A04 | `ticket.approx`: được `false`, muốn `true` (EX-17) | đạt (DONE 1 + DUPLICATE_OPEN_TICKET 1, 1 phiếu, sổ lệnh, audit, `diff` 60, hai vế 2600 / 2540 L) |
| CON-T-819 | A17 | `ticket.approx`: được `false`, muốn `true` (EX-17) | đạt |
| CON-T-820 | A17 | `ticket.approx`: được `false`, muốn `true` (EX-17, gửi hai lần) | đạt (lần hai phát lại) |
| CON-T-821 | A17 | `ticket.approx`: được `false`, muốn `true` (EX-22, chênh lệch âm) | đạt (`diff` −60) |

**Tầng sai đầu tiên: S2 / L1 — chuẩn hoá ô lượng** (`data/normalization/engine.go`, `matchApprox`). Thế giới khai vế kho xuất của EX-17 và
EX-22 là số ước lượng: phiếu kho xlsx, ô lượng `"≈ 2,6"` đơn vị `m3` (`approx: true`, `raw: "≈ 2,6 m3"`). `approx_markers` của pack
(`tenants/construction/pack/lexicon.yaml`) có `"≈"`, nhưng `matchApprox` chỉ nhận một ký hiệu viết cứng trong mã (`"~"`) và trả "không khớp"
cho mọi token ký hiệu khác TRƯỚC khi xét danh sách của pack. Đo trực tiếp (bộ chuẩn hoá của pack, cả chế độ ô lẫn chữ tự do):
`"≈ 2,6 m3"` → `approx=false`; `"≈ 2600 L"` → `approx=false`; `"~ 2,6 m3"` → `approx=true`; `"khoảng 2,6 m3"` → `approx=true`. Đường đi
của cờ sau đó đúng: `quantity()` của đường nạp → fact `Value.Approx = false` → Derive `left.Approx || right.Approx = false` → phiếu
`approx: false`; `Convert` (m3 → L) giữ cờ.

**Không phải lỗi của bộ chạy hay của ca.** Kỳ vọng khớp hợp đồng (phiếu mang `approx` khi một vế dùng số ước lượng) và khớp thế giới; bộ chạy so
đúng trường `approx` của phiếu đọc lại; ba ca A17 còn lại của cùng lớp và mọi lớp khác đạt trên cả hai nền.

**Phạm vi ngoài P1.** Mọi nguồn ghi `"≈"` trước lượng (phiếu kho, báo cáo PDF kiểu "≈ 4 m³", tin nhắn) mất cờ ước lượng ở đường nạp; câu trả
lời của luồng hỏi thiếu ghi chú "ước lượng" cho các số đó. Ảnh hưởng cổng L1 / S2 của CỔNG TRƯỚC AI (M5).

## Hệ quả

- **M4 chưa RA trên bộ giữ lại P1**: X1 và X5 ĐỎ (một lỗi chung ở tầng chuẩn hoá, ngoài mã lệnh); X2, X3, X4, X9 đạt phần khuôn khai được;
  X6, X7 không có ca; X8 đo bằng danh mục đột biến (cần `scripts/mutate.sh` bản có `MUTATE_TEST_MUON`, đi cùng 5c).
- Bộ P1 đã công bố, KHÔNG chạy lại (luật một lần). Sửa `matchApprox` (nhận mọi dấu ký hiệu khai ở `approx_markers`) là một bản sửa thường có
  test đỏ trước; đo lại trên bộ P1 đã công bố chỉ là hồi quy, không phải bằng chứng độc lập. Muốn đo lại M4 độc lập cần bộ giữ lại mới — câu
  cho chủ dự án (Q-P1-15 ở `docs/CHO-QUYET-DINH.md` mục 5).
