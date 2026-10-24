package tenant

import (
	"strings"
	"testing"
)

// F-005: kiểu cha để router gỡ bí danh mơ hồ là khoá narrow_parent của intents.yaml — nạp vào runtime; khai nửa vời (thiếu tên gọi
// hay thiếu kiểu) hoặc trùng tên với cách gỡ theo phạm vi thì chặn lúc nạp.
func TestLoadPackKieuChaThuHep(t *testing.T) {
	rt, err := Load(writePack(t, fullPack(map[string]string{"intents.yaml": validIntents + "narrow_parent: {type: site, label: công trường}\n"})))
	if err != nil {
		t.Fatal(err)
	}
	if p := rt.Router.NarrowParent; p.Type != "site" || p.Label != "công trường" {
		t.Errorf("NarrowParent = %+v", p)
	}
	for name, line := range map[string]string{
		"thiếu tên gọi": "narrow_parent: {type: site}\n",
		"thiếu kiểu":    "narrow_parent: {label: công trường}\n",
		"trùng cách gỡ": "narrow_parent: {type: scope, label: phạm vi}\n",
	} {
		if _, err := Load(writePack(t, fullPack(map[string]string{"intents.yaml": validIntents + line}))); err == nil || !strings.Contains(err.Error(), "narrow_parent") {
			t.Errorf("%s: Load phải lỗi nhắc narrow_parent, được %v", name, err)
		}
	}
}
