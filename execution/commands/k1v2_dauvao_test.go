package commands

// Ca của vòng sửa bước K1 theo bản soát độc lập vòng 1, nhóm đầu vào, quyền, hình kết quả và nạp tool: K1S-02 (số lần đọc danh mục), K1S-07
// (ngày PostgreSQL không có), K1S-09 (ngưỡng âm), K1S-10 (JSON chỉ in lớp lỗi), K1S-11 (UTF-8 hỏng), K1S-12 (thứ tự R4, R5), K1S-13 (tên tool
// chưa cài), K1S-14 (nhãn thật ở bước lỗi). Dùng kho giả, không PostgreSQL, không model.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"unicode/utf8"

	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
)

// ---------------------------------------------------------------------------
// K1S-02: C4 đọc danh mục đúng một lần
// ---------------------------------------------------------------------------

// dirCounter đếm số lần đọc danh mục thực thể; flipAfter > 0: từ lần đọc thứ flipAfter+1 danh mục trả rỗng (giả lập Load nạp lại xen giữa).
type dirCounter struct {
	entityresolution.Directory
	reads     atomic.Int64
	flipAfter int64
}

func (d *dirCounter) Entities(ctx context.Context, types ...string) ([]entityresolution.Entity, error) {
	n := d.reads.Add(1)
	if d.flipAfter > 0 && n > d.flipAfter {
		return nil, nil
	}
	return d.Directory.Entities(ctx, types...)
}

func countedWorld(t *testing.T) (*world, *dirCounter) {
	t.Helper()
	dc := &dirCounter{}
	w := newWorld(t, func(d *Deps) { dc.Directory, d.Directory = d.Directory, dc })
	return w, dc
}

// Số lần đọc danh mục của C4 không được phân biệt máy không tồn tại, máy ngoài phạm vi, khoá có ký tự lạ (§4.4: lệnh là đường ghi, không cho
// dò máy nào tồn tại, kể cả qua độ trễ), và là đúng MỘT lần: đọc một ảnh chụp rồi tính phạm vi trên ảnh đó.
func TestC4DocDanhMucDungMotLanChoMoiLoaiKhoa(t *testing.T) {
	w, dc := countedWorld(t)
	pm := user("p-pm", "project_manager", "PRJ-A")
	keys := map[string]string{
		"không tồn tại": "EX-99", "ngoài phạm vi": "EX-09", "ký tự SQL": "EX-03'; DROP TABLE discrepancy_tickets; --",
		"dấu nháy": `EX-03"`, "khác hoa thường": "ex-03", "trong phạm vi": "EX-03",
	}
	i := 0
	for name, key := range keys {
		i++
		before := dc.reads.Load()
		w.run(pm, req(1100+i, key, day1))
		if n := dc.reads.Load() - before; n != 1 {
			t.Errorf("%s: C4 đọc danh mục %d lần, muốn đúng 1", name, n)
		}
	}
}

// Một lần đọc thì không còn khe giữa hai lần đọc: Load nạp lại xen giữa không làm phạm vi tính trên danh mục rỗng. Ở bản đọc hai lần, lần đọc
// thứ hai (để dựng tổ tiên) thấy danh mục rỗng và từ chối nhầm một máy trong phạm vi.
func TestC4TinhPhamViTrenCungAnhChupVoiLanDocDanhMuc(t *testing.T) {
	w, dc := countedWorld(t)
	dc.flipAfter = 1
	pm := user("p-pm", "project_manager", "PRJ-A")
	wantDone(t, w.run(pm, req(1110, "EX-03", day1)), false)
}

// ---------------------------------------------------------------------------
// K1S-14: nhãn thật của thành phần hỏng ở Step.Fault
// ---------------------------------------------------------------------------

// Bước lỗi mang nhãn thật của thành phần hỏng khi tầng dưới đã gắn (contracts.FaultOf), để bảng vết nói đúng thành phần; Result vẫn giữ nhãn
// của bảng §2.4 (infra:postgres). Không có nhãn thì Step.Fault lấy nhãn của bảng.
func TestStepFaultMangNhanThatCuaThanhPhanHong(t *testing.T) {
	cases := []struct {
		name     string
		arm      func(*world)
		stop     string
		stepFlt  string
		resFault string
	}{
		{"danh mục có nhãn", func(w *world) {
			w.data.dirErr = contracts.Errorf(contracts.Infra("directory"), "danh mục hỏng")
		}, "C4", "infra:directory", "infra:postgres"},
		{"số liệu có nhãn", func(w *world) {
			w.data.factsErr = contracts.Errorf(contracts.Infra("facts"), "số liệu hỏng")
		}, "C5", "infra:facts", "infra:postgres"},
		{"sổ lệnh không nhãn", func(w *world) {
			w.deps.Store = &failReads{Store: w.probe, ledger: errors.New("lỗi thô")}
			w.restart()
		}, "C4b", "infra:postgres", "infra:postgres"},
		{"đơn vị ghi có nhãn", func(w *world) {
			w.deps.Store = &failReads{Store: w.probe, write: contracts.Label(contracts.Infra("pgx"), fmt.Errorf("%w: x", ErrBeforeCommit))}
			w.restart()
		}, "C6", "infra:pgx", "infra:postgres"},
		{"đọc lại có nhãn", func(w *world) {
			w.deps.Store = &readMutator{Store: w.probe, readErr: contracts.Errorf(contracts.Infra("replica"), "đọc lại hỏng")}
			w.restart()
		}, "C7", "infra:replica", "infra:postgres"},
		{"đọc lại không nhãn", func(w *world) {
			w.deps.Store = &readMutator{Store: w.probe, readErr: errors.New("lỗi thô")}
			w.restart()
		}, "C7", "infra:postgres", "infra:postgres"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t)
			c.arm(w)
			o := w.run(user("p-1", "director", ""), req(1120, "EX-03", day1))
			if o.Result.Fault != c.resFault {
				t.Fatalf("Result.Fault = %q, muốn nhãn của bảng §2.4 %q", o.Result.Fault, c.resFault)
			}
			assertSteps(t, o, c.stop)
			if got := o.Steps[codeIndex(t, o, c.stop)].Fault; got != c.stepFlt {
				t.Fatalf("Step.Fault của %s = %q, muốn %q", c.stop, got, c.stepFlt)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// K1S-10: Result ra JSON chỉ mang lớp lỗi
// ---------------------------------------------------------------------------

// Nhãn đầy đủ (ví dụ infra:postgres) không ra khỏi tiến trình qua JSON (§2.3, §2.6.2): trường fault chỉ có lớp lỗi, bên gọi quên gọi
// FaultClass cũng không lộ tên thành phần. Trong tiến trình Fault vẫn đầy đủ để tính mã HTTP.
func TestResultJSONChiInLopLoi(t *testing.T) {
	for code := range reasons {
		r := resultFor(code)
		for how, v := range map[string]any{"giá trị": r, "con trỏ": &r} {
			b, err := json.Marshal(v)
			if err != nil {
				t.Fatal(err)
			}
			var m map[string]any
			if err := json.Unmarshal(b, &m); err != nil {
				t.Fatal(err)
			}
			if m["fault"] != r.FaultClass() || strings.Contains(string(b), "postgres") {
				t.Errorf("%s (%s): JSON phải mang fault=%q và không có tên thành phần: %s", code, how, r.FaultClass(), b)
			}
		}
		if r.Fault != reasons[code].fault {
			t.Errorf("%s: Fault trong tiến trình phải giữ nhãn đầy đủ %q, được %q", code, reasons[code].fault, r.Fault)
		}
	}
	if (Result{Status: StatusDone}).FaultClass() != "" {
		t.Errorf("DONE không có lớp lỗi")
	}
}

// Hình JSON của §2.5 giữ nguyên sau khi thêm MarshalJSON: DONE đủ trường, không có fault và invalid_fields; DUPLICATE_OPEN_TICKET chỉ có mã
// và trạng thái phiếu; INVALID_REQUEST có invalid_fields.
func TestResultJSONGiuHinhMuc25(t *testing.T) {
	w := newWorld(t)
	p := user("p-1", "director", "")
	decode := func(r Result) map[string]any {
		b, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		return m
	}
	keys := func(m map[string]any) []string {
		var out []string
		for k := range m {
			out = append(out, k)
		}
		return out
	}
	hasAll := func(m map[string]any, want ...string) bool {
		for _, k := range want {
			if _, ok := m[k]; !ok {
				return false
			}
		}
		return true
	}

	done := decode(w.run(p, req(1130, "EX-03", day1)).Result)
	if !hasAll(done, "status", "reason_code", "detail", "retry", "replayed", "ticket", "trace_id") || hasAll(done, "fault") || hasAll(done, "invalid_fields") {
		t.Fatalf("JSON của DONE sai hình: %v", keys(done))
	}
	if tk, _ := done["ticket"].(map[string]any); !hasAll(tk, "id", "status", "subject_key", "metric", "day", "left", "right", "diff", "approx") {
		t.Fatalf("ticket của DONE thiếu trường: %v", keys(tk))
	}
	dup := decode(w.run(p, req(1131, "EX-03", day1)).Result)
	if tk, _ := dup["ticket"].(map[string]any); len(tk) != 2 || !hasAll(tk, "id", "status") || dup["fault"] != "data" {
		t.Fatalf("JSON của DUPLICATE_OPEN_TICKET: ticket chỉ có id và status, fault=data: %v", dup)
	}
	bad := decode(w.run(p, Request{}).Result)
	if !hasAll(bad, "invalid_fields") || bad["fault"] != "input" || hasAll(bad, "ticket") {
		t.Fatalf("JSON của INVALID_REQUEST: %v", bad)
	}
}

// ---------------------------------------------------------------------------
// K1S-11: C0 từ chối chuỗi UTF-8 hỏng
// ---------------------------------------------------------------------------

// Chuỗi UTF-8 hỏng làm PostgreSQL từ chối câu INSERT audit hoặc phiếu, nên bị chặn ở C0 như ký tự NUL: INVALID_REQUEST nêu đúng trường,
// không audit; vết của chính lượt bị từ chối vẫn lưu được (mọi chuỗi trong vết là UTF-8 hợp lệ, không có NUL).
func TestC0TuChoiChuoiUTF8Hong(t *testing.T) {
	const bad = "\xff\xfe"
	good := req(1140, "EX-03", day1)
	with := func(f func(*Request)) Request { r := good; f(&r); return r }
	cases := []struct {
		name   string
		req    Request
		fields []string
	}{
		{"subject_key", with(func(r *Request) { r.SubjectKey = "EX-03" + bad }), []string{"subject_key"}},
		{"metric", with(func(r *Request) { r.Metric = "issue_vs_measured" + bad }), []string{"metric"}},
		{"day", with(func(r *Request) { r.Day = day1 + bad }), []string{"day"}},
		{"command_id", with(func(r *Request) { r.CommandID = good.CommandID[:34] + bad }), []string{"command_id"}},
		{"byte thừa đầu chuỗi nhiều byte", with(func(r *Request) { r.SubjectKey = "\x80EX-03" }), []string{"subject_key"}},
		{"chuỗi nhiều byte bị cắt giữa chừng", with(func(r *Request) { r.SubjectKey = "EX-03\xe2\x82" }), []string{"subject_key"}},
		{"ba trường cùng hỏng", with(func(r *Request) { r.SubjectKey, r.Metric, r.Day = bad, bad, bad }), []string{"subject_key", "metric", "day"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newWorld(t)
			o := w.run(user("p-1", "director", ""), c.req)
			wantFail(t, o, StatusFailed, ReasonInvalidRequest)
			if fmt.Sprint(o.Result.InvalidFields) != fmt.Sprint(c.fields) {
				t.Fatalf("invalid_fields = %v, muốn %v", o.Result.InvalidFields, c.fields)
			}
			assertSteps(t, o, "C0")
			for _, s := range o.Steps {
				for _, str := range append(append([]string{s.Decision, s.Reason, s.Error}, s.Input...), s.Output...) {
					if !utf8.ValidString(str) || strings.ContainsRune(str, 0) {
						t.Fatalf("bước %s mang chuỗi UTF-8 hỏng hoặc NUL: %q", s.Code, str)
					}
				}
			}
			if !utf8.ValidString(o.Question) || strings.ContainsRune(o.Question, 0) {
				t.Fatalf("Question mang chuỗi UTF-8 hỏng hoặc NUL: %q", o.Question)
			}
			w.check(tenantA, want{})
			w.counters(0, 0)
		})
	}
	t.Run("UTF-8 hợp lệ nhiều byte không bị chặn ở C0", func(t *testing.T) {
		w := newWorld(t)
		o := w.run(user("p-1", "director", ""), with(func(r *Request) { r.SubjectKey = "Máy xúc 03 – ĐỎ" }))
		wantFail(t, o, StatusFailed, ReasonPermissionDenied) // khoá lạ nhưng hợp lệ về cấu trúc: C4 trả như mọi khoá không tồn tại (§4.4)
		assertSteps(t, o, "C4")
	})
}

// ---------------------------------------------------------------------------
// K1S-07: ngày mà PostgreSQL không biểu diễn được
// ---------------------------------------------------------------------------

// Kiểu date của PostgreSQL không có năm 0: ngày năm 0000 qua được pastDay của Go nhưng làm truy vấn số liệu thật lỗi, và lỗi tất định đó
// sẽ thành STORE_UNAVAILABLE "gửi lại cùng khoá" vô hạn. Chặn ở C3 bằng INVALID_REQUEST [day]. Không thêm cận nghiệp vụ nào khác: ngày xa
// nhưng PostgreSQL biểu diễn được vẫn đi tiếp tới C5 và dừng vì thiếu số liệu.
func TestC3TuChoiNgayNamKhong(t *testing.T) {
	for _, day := range []string{"0000-01-01", "0000-06-15", "0000-12-31"} {
		t.Run("từ chối "+day, func(t *testing.T) {
			w := newWorld(t)
			o := w.run(user("p-1", "director", ""), req(1150, "EX-03", day))
			wantFail(t, o, StatusFailed, ReasonInvalidRequest)
			if fmt.Sprint(o.Result.InvalidFields) != "[day]" {
				t.Fatalf("invalid_fields = %v, muốn [day]", o.Result.InvalidFields)
			}
			assertSteps(t, o, "C3")
			w.check(tenantA, want{})
		})
	}
	for _, day := range []string{"0001-01-01", "1000-05-05", "1900-01-01"} {
		t.Run("không chặn "+day, func(t *testing.T) {
			w := newWorld(t)
			o := w.run(user("p-1", "director", ""), req(1151, "EX-03", day))
			wantFail(t, o, StatusFailed, ReasonInsufficientEvidence) // qua C3, thiếu số liệu của ngày đó
			assertSteps(t, o, "C5")
		})
	}
}

// ---------------------------------------------------------------------------
// K1S-09, K1S-13: cấu hình tool
// ---------------------------------------------------------------------------

// New dùng chung phép kiểm min_abs_diff với LoadTools: ngưỡng âm vô hiệu hoá R4 (chênh lệch 0 vẫn tạo phiếu) nên bị từ chối khi dựng; ngưỡng
// không phải số cũng vậy. Ngưỡng 0, rỗng (mặc định 0) và dương được nhận.
func TestNewTuChoiNguongMinAbsDiffAmHoacKhongPhaiSo(t *testing.T) {
	w := newWorld(t)
	for name, v := range map[string]string{"âm": "-1", "âm thập phân": "-0.5", "không phải số": "abc"} {
		d := w.deps
		d.Tool.MinAbsDiff = v
		if p, err := New(d); err == nil || p != nil || !strings.Contains(err.Error(), "MinAbsDiff") {
			t.Errorf("%s: New phải từ chối và nêu MinAbsDiff, được %v, %v", name, p, err)
		}
	}
	d := w.deps
	d.Tool.MinAbsDiff = "-1"
	if _, err := New(d); err == nil || !strings.Contains(err.Error(), "không được âm") {
		t.Errorf("lời của New cho ngưỡng âm phải là lời chung với LoadTools (không được âm): %v", err)
	}
	for _, v := range []string{"", "0", "0.5", "100"} {
		d := w.deps
		d.Tool.MinAbsDiff = v
		if _, err := New(d); err != nil {
			t.Errorf("ngưỡng %q phải được nhận: %v", v, err)
		}
	}
}

// LoadTools đóng mặc định theo tên: chỉ nạp tool mà gói đã cài. Tên lạ không thành route lệnh và không vào Names(); lỗi nêu tên lạ cùng tên
// đã cài để sửa, và vẫn gom với các lỗi khác của cùng tệp.
func TestLoadToolsTuChoiTenToolChuaCaiTrongGoi(t *testing.T) {
	pol, cat := loadFakePack(t)
	const body = "    capability: tickets.create\n    metrics: [issue_vs_measured]\n"
	cases := []struct {
		name    string
		yaml    string
		wantErr []string
	}{
		{"tên lạ duy nhất", "tools:\n  - name: close_all_tickets\n" + body, []string{"close_all_tickets", "chưa cài", CommandOpenDiscrepancyTicket}},
		{"tên đúng cùng tên lạ", "tools:\n  - name: open_discrepancy_ticket\n" + body + "  - name: close_all_tickets\n" + body, []string{"close_all_tickets", "chưa cài"}},
		{"khác hoa thường", "tools:\n  - name: Open_Discrepancy_Ticket\n" + body, []string{"Open_Discrepancy_Ticket", "chưa cài"}},
		{"gom với lỗi khác", "tools:\n  - name: close_all_tickets\n    metrics: [issue_vs_measured]\n", []string{"close_all_tickets", "chưa cài", "capability"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tools, err := LoadTools(packWith(t, c.yaml), pol, cat)
			if err == nil || tools != nil {
				t.Fatalf("tên tool chưa cài phải làm nạp lỗi và không trả tập nào, được %v, %v", tools, err)
			}
			for _, w := range c.wantErr {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("lỗi %q thiếu %q", err.Error(), w)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// K1S-12: thứ tự R4 trước R5 (hợp đồng chưa đặc tả)
// ---------------------------------------------------------------------------

// HOP-DONG §5 chỉ cố định R3 (xung đột trước thiếu vế); thứ tự R4 (có chênh lệch) và R5 (không trùng) CHƯA được đặc tả. Bản cài chọn R4
// trước: khi cả hai cùng đúng, NO_DISCREPANCY thắng DUPLICATE_OPEN_TICKET. Ca này ghim lựa chọn đó để đổi thứ tự phải là việc có chủ ý
// (câu hỏi cho chủ hợp đồng), không phải hệ quả ngầm của một lần sửa.
func TestR4DungTruocR5KhiCaHaiCungDung(t *testing.T) {
	w := newWorld(t)
	p := user("p-1", "director", "")
	wantDone(t, w.run(p, req(1160, "EX-03", day1)), false) // phiếu OPEN sẵn cho (EX-03, ngày), chênh lệch 68
	strict := w.deps
	strict.Tool.MinAbsDiff = "100" // chênh lệch 68 nay không vượt ngưỡng
	pipe, err := New(strict)
	if err != nil {
		t.Fatal(err)
	}
	o := pipe.Run(context.Background(), p, req(1161, "EX-03", day1)) // khác khoá: trùng nghiệp vụ VÀ không vượt ngưỡng
	wantFail(t, o, StatusFailed, ReasonNoDiscrepancy)
	assertSteps(t, o, "C5")
}
