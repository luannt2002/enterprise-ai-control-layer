-- 0011 — ghi chú suy luận của fact (F-008 của báo cáo audit vNext, phần còn lại). Con số gắn thực thể bằng SUY LUẬN lúc nạp (nguồn
-- carry_entity: "đơn hàng kế thừa từ tin trước của Lan cùng ngày") mang lời ghi chú đó vào facts, để truy được số nào là suy luận.
-- Trước bản này ghi chú chỉ có ở quan sát của bộ nạp và mất khi thành fact.
-- Cột có thể NULL: NULL = số đọc thẳng từ nguồn (mọi dòng đã có); thêm cột NULL không giá trị mặc định không viết lại bảng.
-- Quyền mức bảng của vai ứng dụng (0002) và chính sách RLS phủ luôn cột mới.
-- Không sửa migration cũ (ADR-0017). Chạy lại được: ADD COLUMN IF NOT EXISTS.
ALTER TABLE facts ADD COLUMN IF NOT EXISTS note text;
