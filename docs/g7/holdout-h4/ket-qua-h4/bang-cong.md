| Lớp | đạt ở cả hai đường / tổng | bộ nhớ | PG |
|---|---|---|---|
| representation | 16/18 | 16/18 | 16/18 |
| entity | 16/16 | 16/16 | 16/16 |
| ambiguity | 13/14 | 13/14 | 13/14 |
| semantic | 13/14 | 13/14 | 13/14 |
| source_conflict | 12/15 | 12/15 | 12/15 |
| permission | 16/16 | 16/16 | 16/16 |
| unsupported_answer | 9/15 | 9/15 | 9/15 |
| routing | 9/11 | 9/11 | 9/11 |

- số ca trong tệp: 119; authored_by independent: 119/119
- phát lại (câu có bản ghi quyết định, lệch): bộ nhớ (119, 0), PG (119, 0); cần (119, 0) ở cả hai
- câu trả lời chứa canary: 0 ca 
- ca đỏ (ở ít nhất một đường): 15 WIN-I-705[representation bộ-nhớ PG] WIN-I-716[representation bộ-nhớ PG] WIN-I-740[ambiguity bộ-nhớ PG] WIN-I-758[semantic bộ-nhớ PG] WIN-I-806[source_conflict bộ-nhớ PG] WIN-I-808[source_conflict bộ-nhớ PG] WIN-I-810[source_conflict bộ-nhớ PG] WIN-I-835[unsupported_answer bộ-nhớ PG] WIN-I-837[unsupported_answer bộ-nhớ PG] WIN-I-838[unsupported_answer bộ-nhớ PG] WIN-I-841[unsupported_answer bộ-nhớ PG] WIN-I-847[unsupported_answer bộ-nhớ PG] WIN-I-848[unsupported_answer bộ-nhớ PG] WIN-I-854[routing bộ-nhớ PG] WIN-I-862[routing bộ-nhớ PG]
- hai đường cho cùng nhãn + cùng câu trả lời từng ca (báo riêng, không thuộc G6'): True
- độ nhạy 1 (đăng ký trước): ca có ký tự nhìn giống trong câu hỏi: 0, đạt 0/0; representation khi bỏ các ca đó: 16/18
- độ nhạy 3 (ADR-0006 §10, đăng ký trước H3): ca xác nhận 42, kỳ vọng hỏi lại / thiếu bằng chứng 6; ĐỎ chỉ vì trả số không phán quyết — chặt 0: -; nới (kèm route kéo theo) 0: -

G2' phần ca (entity + ambiguity 100%, cả hai đường): KHÔNG ĐẠT
G5' phần ca (permission + unsupported_answer 100%, cả hai đường, 0 canary): KHÔNG ĐẠT
G6' phần đo được ở đây (≥ 100 ca, mỗi lớp ≥ 5, phát lại 100% và 0 lệch ở cả hai đường, 100% độc lập): ĐẠT
(G6' còn cần: đóng băng + push TRƯỚC lần chạy đầu — đối chiếu giờ push với giờ chạy; đột biến mỗi cổng; test thuộc tính. G2' còn cần kiểm 'đúng tập ứng viên' ở ca ambiguity có khai candidates.)
