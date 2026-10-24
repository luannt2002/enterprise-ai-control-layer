package normalization

import "testing"

// F-030: danh từ nhãn / đếm của NGÀNH ("khối 2", "L 5", "2 máy", "3-4 chuyến") là của pack (lexicon: label_nouns, count_nouns), không viết cứng
// trong bộ chuẩn hoá dùng chung. Pack không khai thì con số đứng cạnh chúng vẫn là một con số (hướng an toàn: luồng hỏi hỏi lại, không phán
// quyết); từ chức năng ("phiếu 912", "3 lần", "mỗi ngày 3") là nhãn với mọi pack. Danh từ nhãn của pack cũng là bí danh đơn vị thì đứng sau một
// chữ số nó là đơn vị của số đó ("3850 khối 19": 19 không phải nhãn); danh từ nhãn không phải đơn vị ("bể") thì vẫn là nhãn.
func TestDanhTuNhanDemDoPackKhai(t *testing.T) {
	bare := testLexicon
	bare.LabelNouns, bare.CountNouns = nil, nil
	none, err := Compile(bare)
	if err != nil {
		t.Fatal(err)
	}
	other := testLexicon
	other.LabelNouns, other.CountNouns = []string{"bể"}, []string{"đơn"}
	own, err := Compile(other)
	if err != nil {
		t.Fatal(err)
	}
	pack := mustEngine(t)
	for _, c := range []struct {
		in, num         string
		pack, none, own bool
	}{
		{"cho 2 máy", "2", true, false, false},
		{"1 xe", "1", true, false, false},
		{"đổ 3-4 chuyến", "3", true, false, false},
		{"2 thiết bị", "2", true, false, false},
		{"3 dự án", "3", true, false, false},
		{"khối 2", "2", true, false, false},
		{"L 5", "5", true, false, false},
		{"lít 5", "5", true, false, false},
		{"3850 khối 19", "19", false, false, false},
		{"bể 7", "7", false, false, true},
		{"3 bể 7", "7", false, false, true},
		{"2 đơn", "2", false, false, true},
		{"theo phiếu 912", "912", true, true, true},
		{"đổ 3 lần", "3", true, true, true},
		{"mỗi ngày 3 bể", "3", true, true, true},
		{"được cấp 3850", "3850", false, false, false},
	} {
		start := indexOf(c.in, c.num)
		for _, k := range []struct {
			name string
			e    *Engine
			want bool
		}{{"pack đủ", pack, c.pack}, {"pack không khai", none, c.none}, {"pack khác", own, c.own}} {
			if got := k.e.LabelContext(c.in, start, start+len(c.num)); got != k.want {
				t.Errorf("%s: LabelContext(%q, %q) = %v, muốn %v", k.name, c.in, c.num, got, k.want)
			}
		}
	}
	// danh từ rỗng sẽ khớp mọi con số: lexicon hỏng, không được chạy
	for _, lex := range []Lexicon{{LabelNouns: []string{" "}}, {CountNouns: []string{""}}} {
		if _, err := Compile(lex); err == nil {
			t.Errorf("Compile(%+v) nhận danh từ rỗng, muốn lỗi", lex)
		}
	}
}
