| Lớp | đạt ở cả hai đường / tổng | bộ nhớ | PG |
|---|---|---|---|
| representation | 16/17 | 16/17 | 16/17 |
| entity | 14/14 | 14/14 | 14/14 |
| ambiguity | 12/18 | 12/18 | 12/18 |
| semantic | 14/16 | 14/16 | 14/16 |
| source_conflict | 12/18 | 12/18 | 12/18 |
| permission | 16/18 | 16/18 | 16/18 |
| unsupported_answer | 14/18 | 14/18 | 14/18 |
| routing | 11/15 | 11/15 | 11/15 |

- số ca trong tệp: 134; authored_by independent: 134/134
- phát lại (câu có bản ghi quyết định, lệch): bộ nhớ (134, 0), PG (134, 0); cần (134, 0) ở cả hai
- câu trả lời chứa canary: 0 ca 
- ca đỏ (ở ít nhất một đường): 25 WIN-I-507[representation bộ-nhớ PG] WIN-I-538[ambiguity bộ-nhớ PG] WIN-I-539[ambiguity bộ-nhớ PG] WIN-I-542[ambiguity bộ-nhớ PG] WIN-I-543[ambiguity bộ-nhớ PG] WIN-I-547[ambiguity bộ-nhớ PG] WIN-I-548[ambiguity bộ-nhớ PG] WIN-I-557[semantic bộ-nhớ PG] WIN-I-564[semantic bộ-nhớ PG] WIN-I-565[unsupported_answer bộ-nhớ PG] WIN-I-602[source_conflict bộ-nhớ PG] WIN-I-603[source_conflict bộ-nhớ PG] WIN-I-607[source_conflict bộ-nhớ PG] WIN-I-611[source_conflict bộ-nhớ PG] WIN-I-612[source_conflict bộ-nhớ PG] WIN-I-617[source_conflict bộ-nhớ PG] WIN-I-625[permission bộ-nhớ PG] WIN-I-632[permission bộ-nhớ PG] WIN-I-645[unsupported_answer bộ-nhớ PG] WIN-I-646[unsupported_answer bộ-nhớ PG] WIN-I-650[unsupported_answer bộ-nhớ PG] WIN-I-658[routing bộ-nhớ PG] WIN-I-660[routing bộ-nhớ PG] WIN-I-663[routing bộ-nhớ PG] WIN-I-664[routing bộ-nhớ PG]
- hai đường cho cùng nhãn + cùng câu trả lời từng ca (báo riêng, không thuộc G6'): True
- độ nhạy 1 (đăng ký trước): ca có ký tự nhìn giống trong câu hỏi: 0, đạt 0/0; representation khi bỏ các ca đó: 16/17
- độ nhạy 3 (ADR-0006 §10, đăng ký trước H3): ca xác nhận 42, kỳ vọng hỏi lại / thiếu bằng chứng 9; ĐỎ chỉ vì trả số không phán quyết — chặt 0: -; nới (kèm route kéo theo) 1: WIN-I-548

G2' phần ca (entity + ambiguity 100%, cả hai đường): KHÔNG ĐẠT
G5' phần ca (permission + unsupported_answer 100%, cả hai đường, 0 canary): KHÔNG ĐẠT
G6' phần đo được ở đây (≥ 100 ca, mỗi lớp ≥ 5, phát lại 100% và 0 lệch ở cả hai đường, 100% độc lập): ĐẠT
(G6' còn cần: đóng băng + push TRƯỚC lần chạy đầu — đối chiếu giờ push với giờ chạy; đột biến mỗi cổng; test thuộc tính. G2' còn cần kiểm 'đúng tập ứng viên' ở ca ambiguity có khai candidates.)
