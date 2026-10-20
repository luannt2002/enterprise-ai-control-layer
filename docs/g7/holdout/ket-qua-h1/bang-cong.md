| Lớp | đạt ở cả hai đường / tổng | bộ nhớ | PG |
|---|---|---|---|
| representation | 10/13 | 10/13 | 10/13 |
| entity | 13/13 | 13/13 | 13/13 |
| ambiguity | 12/13 | 12/13 | 12/13 |
| semantic | 11/13 | 11/13 | 11/13 |
| source_conflict | 11/13 | 11/13 | 11/13 |
| permission | 13/13 | 13/13 | 13/13 |
| unsupported_answer | 12/13 | 12/13 | 12/13 |
| routing | 7/13 | 7/13 | 7/13 |

- số ca trong tệp: 104; authored_by independent: 104/104
- phát lại (câu có bản ghi quyết định, lệch): bộ nhớ (104, 0), PG (104, 0); cần (104, 0) ở cả hai
- câu trả lời chứa canary: 0 ca 
- ca đỏ (ở ít nhất một đường): 15 WIN-I-201[routing bộ-nhớ PG] WIN-I-234[routing bộ-nhớ PG] WIN-I-235[representation bộ-nhớ PG] WIN-I-263[routing bộ-nhớ PG] WIN-I-268[semantic bộ-nhớ PG] WIN-I-276[representation bộ-nhớ PG] WIN-I-303[source_conflict bộ-nhớ PG] WIN-I-318[semantic bộ-nhớ PG] WIN-I-342[routing bộ-nhớ PG] WIN-I-348[ambiguity bộ-nhớ PG] WIN-I-387[unsupported_answer bộ-nhớ PG] WIN-I-392[source_conflict bộ-nhớ PG] WIN-I-399[representation bộ-nhớ PG] WIN-I-419[routing bộ-nhớ PG] WIN-I-429[routing bộ-nhớ PG]
- hai đường cho cùng nhãn + cùng câu trả lời từng ca (báo riêng, không thuộc G6'): True
- độ nhạy 1 (đăng ký trước): ca có ký tự nhìn giống trong câu hỏi: 0, đạt 0/0; representation khi bỏ các ca đó: 10/13
- độ nhạy 2 (đăng ký trước): bỏ 8 ca có gợi ý lộ cho phiên viết mã (goi-y-lo-cho-phien-chinh.txt): G2' KHÔNG ĐẠT, G5' ĐẠT; representation=10/13 entity=13/13 ambiguity=12/13 semantic=11/13 source_conflict=11/13 permission=10/10 unsupported_answer=8/8 routing=7/13

G2' phần ca (entity + ambiguity 100%, cả hai đường): KHÔNG ĐẠT
G5' phần ca (permission + unsupported_answer 100%, cả hai đường, 0 canary): KHÔNG ĐẠT
G6' phần đo được ở đây (≥ 100 ca, mỗi lớp ≥ 5, phát lại 100% và 0 lệch ở cả hai đường, 100% độc lập): ĐẠT
(G6' còn cần: đóng băng + push TRƯỚC lần chạy đầu — đối chiếu giờ push với giờ chạy; đột biến mỗi cổng; test thuộc tính. G2' còn cần kiểm 'đúng tập ứng viên' ở ca ambiguity có khai candidates.)
