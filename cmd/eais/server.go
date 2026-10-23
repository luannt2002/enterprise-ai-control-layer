package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/core/pipeline"
	"github.com/luannt2002/enterprise-ai-control-layer/observability/traces"
	"github.com/luannt2002/enterprise-ai-control-layer/platform/pgstore"
)

const (
	maxAskBody    = 64 << 10 // trần thân /v1/ask: câu hỏi tối đa pipeline.MaxTextRunes ký tự, còn lại là vài trường ngắn
	maxIngestBody = 1 << 10  // thân /v1/ingest phải rỗng hoặc {}; trần này chỉ để không đọc vô hạn
	saveTimeout   = 5 * time.Second
	ingestTimeout = 15 * time.Minute
)

// counters là bộ đếm của những gì KHÔNG có dòng vết theo tenant hoặc thất bại ngoài vết (xem platform/pgstore/vet.go: lượt chưa xác
// thực được tenant nằm ngoài kho vết nên cần bộ đếm riêng; chỉ tiêu "mọi lượt có vết" tính trên lượt đã có tenant). In ra log khi
// tắt server. Mọi bộ đếm "thất bại" phải bằng 0 khi hệ khoẻ.
type counters struct {
	Unauthenticated atomic.Int64 // thiếu / sai / lạ token: không có tenant ⇒ không có dòng vết
	TraceSaveFailed atomic.Int64 // lượt ĐÃ xác thực mà không lưu được vết (sau khi thử lại): vi phạm chỉ tiêu, phải là 0
	UsageSaveFailed atomic.Int64
	IngestDenied    atomic.Int64 // yêu cầu nạp lại bị từ chối vì vai
	HostRejected    atomic.Int64 // tiêu đề Host không phải loopback (chống DNS rebinding)
}

// server là lớp HTTP của app. Chỉ GỌI pipeline.Ask.Handle như hiện có: không chỉnh câu trả lời, không thêm luật nào vào luồng hỏi.
type server struct {
	app        *app
	st         store
	log        *slog.Logger
	principals map[string]contracts.Principal // token → principal (chế độ dev: suy từ world.yaml)
	askTimeout time.Duration
	hostGuard  bool // chế độ dev: Host phải là loopback
	cnt        counters
	ingestMu   sync.Mutex
}

func newServer(a *app, principals map[string]contracts.Principal, log *slog.Logger) *server {
	return &server{app: a, st: a.st, log: log, principals: principals, askTimeout: 30 * time.Second}
}

// errBody là thân mọi phản hồi lỗi. trace_id chỉ có khi vết của lượt đã lưu bền được.
type errBody struct {
	Error   string `json:"error"`
	Fault   string `json:"fault,omitempty"`
	TraceID string `json:"trace_id,omitempty"`
}

// askRequest là thân POST /v1/ask. KHÔNG có trường tenant / principal / vai: những thứ đó chỉ đến từ token (CLAUDE.md §11, §21).
type askRequest struct {
	Text           string `json:"text"`
	ConversationID string `json:"conversation_id"`
	MessageID      string `json:"message_id"`
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.healthz)
	mux.HandleFunc("POST /v1/ask", s.authed(s.ask))
	mux.HandleFunc("POST /v1/ingest", s.authed(s.ingest))
	mux.HandleFunc("GET /v1/trace/{id}", s.authed(s.traceGet))
	return s.logged(s.guardHost(mux))
}

// statusWriter nhớ mã trạng thái cho nhật ký truy cập; Unwrap để http.ResponseController đặt được hạn ghi.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// logged ghi một dòng mỗi yêu cầu: phương thức, đường dẫn, mã, thời gian, trace_id. KHÔNG bao giờ ghi tiêu đề Authorization hay thân.
func (s *server) logged(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		next.ServeHTTP(sw, r)
		if r.URL.Path == "/healthz" && sw.status == http.StatusOK {
			return // thăm dò sức khoẻ thành công không đáng một dòng log
		}
		s.log.Info("yêu cầu", "method", r.Method, "path", r.URL.Path, "status", sw.status, "ms", time.Since(start).Milliseconds(), "trace_id", w.Header().Get("X-Trace-Id"))
	})
}

// guardHost chặn Host không phải loopback ở chế độ dev: token dev là công khai, nên một trang web ngoài trỏ tên của nó về 127.0.0.1 (DNS
// rebinding) không được gọi API này từ trình duyệt của người dùng.
func (s *server) guardHost(next http.Handler) http.Handler {
	if !s.hostGuard {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !loopbackHost(r.Host) {
			s.cnt.HostRejected.Add(1)
			s.log.Warn("từ chối Host không phải loopback (chế độ dev)", "host", traces.Sanitize(r.Host), "remote", r.RemoteAddr)
			writeJSON(w, http.StatusForbidden, errBody{Error: "Host không phải loopback (chế độ dev)", Fault: contracts.FaultPolicy})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// authed bọc handler cần principal. Principal lấy từ token ở tiêu đề Authorization — KHÔNG từ thân, KHÔNG từ query. Thiếu / sai / lạ
// token cùng một phản hồi 401 (không để dò xem token nào tồn tại); lý do thật chỉ nằm trong log. Lượt này không có tenant nên không
// có dòng vết: tăng bộ đếm Unauthenticated (platform/pgstore/vet.go giải thích vì sao).
func (s *server) authed(h func(http.ResponseWriter, *http.Request, contracts.Principal)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		reason := "thiếu hoặc sai dạng tiêu đề Authorization"
		if tok, ok := bearerToken(r.Header.Get("Authorization")); ok {
			if pr, found := s.principals[tok]; found {
				h(w, r, clonePrincipal(pr))
				return
			}
			reason = "token lạ"
		}
		s.cnt.Unauthenticated.Add(1)
		s.log.Warn("chưa xác thực", "lý do", reason, "method", r.Method, "path", r.URL.Path, "remote", r.RemoteAddr)
		w.Header().Set("WWW-Authenticate", `Bearer realm="eais"`)
		writeJSON(w, http.StatusUnauthorized, errBody{Error: "chưa xác thực: cần tiêu đề Authorization: Bearer <token>", Fault: contracts.FaultPolicy})
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

// deadline đặt hạn ghi phản hồi riêng cho handler (server không đặt WriteTimeout chung vì nạp lại dữ liệu chạy lâu hơn một lượt hỏi).
func deadline(w http.ResponseWriter, d time.Duration) {
	_ = http.NewResponseController(w).SetWriteDeadline(time.Now().Add(d))
}

// isCallerFault: lỗi do bên gọi (input) hoặc do chính sách (policy) — thông điệp của chúng viết cho người gọi.
func isCallerFault(fault string) bool {
	return fault == contracts.FaultInput || fault == contracts.FaultPolicy
}

// publicMessage là thông điệp lỗi giao cho client. Lỗi do bên gọi (input) và do chính sách (policy) có thông điệp viết cho người
// gọi nên giữ nguyên; mọi lỗi khác có thể mang chi tiết nội bộ (lỗi driver, đường dẫn…) nên client chỉ nhận câu chung; chi tiết đã
// làm sạch nằm trong vết (nếu lượt có vết) và trong log. Câu này KHÔNG nhắc trace_id: chỉ failTurn biết lượt có vết hay không.
func publicMessage(fault, msg string) string {
	if isCallerFault(fault) {
		return msg
	}
	return "lỗi hệ thống (chi tiết nội bộ không công bố qua API)"
}

// publicFault là nhãn lỗi giao cho client: chỉ LỚP lỗi ("infra", "third_party", "code"…), không có tên thành phần nội bộ phía sau dấu
// hai chấm ("infra:postgres" → "infra"). Nhãn đầy đủ nằm trong vết, cho người vận hành.
func publicFault(fault string) string {
	class, _, _ := strings.Cut(fault, ":")
	return class
}

func (s *server) healthz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.st.Ping(ctx); err != nil {
		fault, msg := traces.SanitizeError(err)
		s.log.Error("healthz: kho không trả lời", "fault", fault, "lỗi", msg)
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded", "reason": "postgres"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ask xử lý một lượt hỏi. MỌI lượt đã xác thực đều để lại vết bền: lượt chạy xong, lượt Handle trả lỗi (kể cả panic), lượt bị chính
// sách chặn, lượt thân sai. Lượt chạy xong mà không lưu được vết thì KHÔNG giao câu trả lời (đóng mặc định): câu trả lời không truy
// được thì không phải câu trả lời của một hệ P8.
func (s *server) ask(w http.ResponseWriter, r *http.Request, pr contracts.Principal) {
	deadline(w, 2*s.askTimeout)
	req, status, err := decodeAsk(w, r)
	if err != nil {
		s.failTurn(w, r, pr, "", err, status, true, nil)
		return
	}
	acc := &usageAcc{}
	ctx, cancel := context.WithTimeout(context.WithValue(r.Context(), usageKey{}, acc), s.askTimeout)
	defer cancel()
	out, err := s.handle(ctx, pr, contracts.AskRequest{ConversationID: req.ConversationID, MessageID: req.MessageID, Text: req.Text})
	if err != nil {
		s.failTurn(w, r, pr, req.Text, err, 0, false, acc)
		return
	}
	// Luồng hỏi tự ghi phiên bản mã riêng của nó vào Decision, và là "(devel)" khi binary không mang thông tin VCS. Ứng dụng là nơi biết binary
	// nào đang chạy nên ghi đè bằng phiên bản của mình: vết lượt thành công, vết lượt lỗi (versions) và mã lần nạp cùng một nguồn. Đặt trước
	// NewRecord vì từng bước của vết chép phiên bản từ Decision.
	out.Decision.CodeVersion = buildRevision()
	rec := traces.NewRecord(traces.Turn{Decision: out.Decision, Spans: out.Spans, TotalMs: out.TotalMs,
		UnaccountedMs: out.UnaccountedMs, Plan: out.Plan, Response: out.Response})
	if err := s.persist(r.Context(), pr, rec, acc); err != nil {
		fault, _ := traces.SanitizeError(err)
		status := contracts.HTTPStatus(err)
		// bản ghi do CHÍNH server dựng bị từ chối — vì đầu vào / chính sách, hay vì PG trả lỗi dữ liệu (lớp 22 / 23, có SQLSTATE): lỗi của server,
		// không phải của client. Xung đột nội dung do kho tự phát hiện (cùng trace_id, khác nội dung) vẫn theo nhãn: 409.
		if fault == contracts.FaultInput || fault == contracts.FaultPolicy || (fault == contracts.FaultData && pgstore.SQLState(err) != "") {
			status = http.StatusInternalServerError
		}
		writeJSON(w, status, errBody{Error: "không lưu được vết của lượt này nên không giao câu trả lời (không có trace_id; xem log của server)", Fault: publicFault(fault)})
		return
	}
	w.Header().Set("X-Trace-Id", out.Response.TraceID)
	writeJSON(w, http.StatusOK, out.Response)
}

// handle gọi Handle, biến panic thành lỗi của mã để lượt vẫn có vết (net/http chỉ đóng kết nối và không ai biết lượt đó đã hỏng).
func (s *server) handle(ctx context.Context, pr contracts.Principal, req contracts.AskRequest) (out pipeline.Outcome, err error) {
	defer func() {
		if p := recover(); p != nil {
			s.log.Error("panic trong luồng hỏi", "panic", traces.Sanitize(fmt.Sprint(p)), "stack", string(debug.Stack()))
			out, err = pipeline.Outcome{}, contracts.Errorf(contracts.FaultCode, fmt.Sprintf("panic trong luồng hỏi: %v", p))
		}
	}()
	return s.app.ask.Load().Handle(ctx, pr, req)
}

// failTurn ghi vết của một lượt lỗi rồi trả lỗi cho client. status = 0 thì suy từ nhãn lỗi (contracts.HTTPStatus). atAccept: lỗi xảy
// ra ở S0 của lớp HTTP (thân sai) — biết chắc bước nào nên ghi đúng S0 thay vì "chưa biết" (S?).
func (s *server) failTurn(w http.ResponseWriter, r *http.Request, pr contracts.Principal, question string, err error, status int, atAccept bool, acc *usageAcc) {
	traceID := uuid.NewString()
	rec := traces.NewErrorRecord(traceID, pr, question, s.app.now(), s.app.versions(), err)
	if atAccept {
		st := &rec.Steps[0]
		st.Stage, st.Name = traces.StageAccept, "accept"
		st.Reason = "thân yêu cầu bị từ chối ở lớp HTTP trước khi vào luồng hỏi"
	}
	if status == 0 {
		status = contracts.HTTPStatus(err)
	}
	body := errBody{Error: publicMessage(rec.Fault, rec.Error), Fault: publicFault(rec.Fault)}
	if s.persist(r.Context(), pr, rec, acc) == nil {
		body.TraceID = traceID
		w.Header().Set("X-Trace-Id", traceID)
		if !isCallerFault(rec.Fault) {
			body.Error += "; tra cứu vết bằng trace_id"
		}
	} else if !isCallerFault(rec.Fault) {
		body.Error += "; vết của lượt này cũng không lưu được nên không có trace_id (xem log của server)"
	}
	writeJSON(w, status, body)
}

// persist lưu bản ghi vết rồi usage của MỘT lượt. Dùng ngữ cảnh không huỷ theo client (WithoutCancel) cùng hạn riêng: client ngắt
// kết nối giữa chừng không được làm mất vết của lượt đã chạy. Thử lại CHÍNH bản ghi đó khi lỗi hạ tầng tạm thời (hợp đồng ở
// platform/pgstore/vet.go: dựng bản ghi một lần, thử lại bằng chính nó). Lỗi lưu vết trả về; lỗi lưu usage chỉ đếm và log (vết đã
// có, usage của model extractive là 0).
func (s *server) persist(ctx context.Context, pr contracts.Principal, rec traces.Record, acc *usageAcc) error {
	sctx, cancel := context.WithTimeout(context.WithoutCancel(contracts.WithTenant(ctx, pr.TenantID)), saveTimeout)
	defer cancel()
	if err := retry(sctx, func() error { _, err := s.st.SaveDecision(sctx, rec); return err }); err != nil {
		s.cnt.TraceSaveFailed.Add(1)
		fault, msg := traces.SanitizeError(err)
		s.log.Error("KHÔNG lưu được vết", "trace_id", rec.Decision.TraceID, "tenant", pr.TenantID, "fault", fault, "lỗi", msg)
		return err
	}
	if acc == nil {
		acc = &usageAcc{}
	}
	u := acc.record(rec.Decision.TraceID, s.app.model.name)
	if err := retry(sctx, func() error { _, err := s.st.SaveUsage(sctx, u); return err }); err != nil {
		s.cnt.UsageSaveFailed.Add(1)
		fault, msg := traces.SanitizeError(err)
		s.log.Error("KHÔNG lưu được usage", "trace_id", rec.Decision.TraceID, "tenant", pr.TenantID, "fault", fault, "lỗi", msg)
	}
	return nil
}

// retry chạy fn tối đa 3 lần, chỉ thử lại lỗi hạ tầng tạm thời (contracts.Retryable), nghỉ 50 / 100 ms giữa các lần.
func retry(ctx context.Context, fn func() error) error {
	var err error
	for attempt := 1; attempt <= 3; attempt++ {
		if err = fn(); err == nil || !contracts.Retryable(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return err
		case <-time.After(time.Duration(attempt) * 50 * time.Millisecond):
		}
	}
	return err
}

func inputErr(msg string) error { return contracts.Errorf(contracts.FaultInput, msg) }

// decodeAsk đọc thân /v1/ask một cách NGHIÊM: đúng application/json, có trần kích thước, trường lạ bị từ chối (kể cả mọi trường kiểu
// tenant / principal / vai — định danh chỉ đến từ token), không có nội dung thừa sau đối tượng. Trả mã HTTP khi lỗi.
func decodeAsk(w http.ResponseWriter, r *http.Request) (askRequest, int, error) {
	var req askRequest
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		return req, http.StatusUnsupportedMediaType, inputErr("Content-Type phải là application/json")
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxAskBody))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		var tooBig *http.MaxBytesError
		switch {
		case errors.As(err, &tooBig):
			return req, http.StatusRequestEntityTooLarge, inputErr(fmt.Sprintf("thân yêu cầu quá %d byte", maxAskBody))
		case errors.Is(err, io.EOF):
			return req, http.StatusBadRequest, inputErr("thân yêu cầu rỗng")
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			return req, http.StatusBadRequest, inputErr(unknownFieldMessage(strings.Trim(strings.TrimPrefix(err.Error(), "json: unknown field "), `"`)))
		}
		return req, http.StatusBadRequest, inputErr("thân yêu cầu không phải JSON hợp lệ: " + err.Error())
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return req, http.StatusBadRequest, inputErr("thân yêu cầu có nội dung thừa sau đối tượng JSON")
	}
	return req, 0, nil
}

// unknownFieldMessage nói rõ vì sao một trường lạ bị từ chối; trường nhắm định danh được nêu luật riêng.
func unknownFieldMessage(name string) string {
	lower := strings.ToLower(name)
	for _, k := range []string{"tenant", "principal", "role", "scope", "user", "token", "auth"} {
		if strings.Contains(lower, k) {
			return fmt.Sprintf("thân yêu cầu không được mang %q: tenant, principal, vai và phạm vi chỉ lấy từ token, không bao giờ từ thân", name)
		}
	}
	return fmt.Sprintf("trường lạ %q: thân chỉ nhận text, conversation_id, message_id", name)
}

// fullAccess: principal của đúng tenant mà tiến trình này phục vụ VÀ có toàn quyền dữ liệu theo policy của pack (xem fullAccess ở
// auth.go). Nạp lại THAY toàn bộ dữ liệu của tenant nên chỉ họ được nạp; họ cũng là những người đọc được vết của người khác.
func (s *server) fullAccess(pr contracts.Principal) bool {
	return string(pr.TenantID) == s.app.tenantID() && fullAccess(s.app.rt.Policy, pr)
}

// ingest nạp lại thư mục dữ liệu ĐÃ CẤU HÌNH của tenant của principal. Thân không được mang gì (rỗng hoặc {}): KHÔNG nhận đường
// dẫn, tên tenant hay tham số nào từ client.
func (s *server) ingest(w http.ResponseWriter, r *http.Request, pr contracts.Principal) {
	deadline(w, ingestTimeout+time.Minute)
	if !s.fullAccess(pr) {
		s.cnt.IngestDenied.Add(1)
		s.log.Warn("từ chối nạp lại dữ liệu: vai không đủ quyền", "tenant", pr.TenantID, "principal", pr.ID, "roles", pr.Roles)
		writeJSON(w, http.StatusForbidden, errBody{Error: "vai của bạn không được phép nạp lại dữ liệu", Fault: contracts.FaultPolicy})
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxIngestBody))
	if t := bytes.TrimSpace(body); err != nil || (len(t) != 0 && string(t) != "{}") {
		writeJSON(w, http.StatusBadRequest, errBody{Error: "thân yêu cầu phải rỗng hoặc {}: nạp lại dùng thư mục dữ liệu đã cấu hình, không nhận đường dẫn hay tham số", Fault: contracts.FaultInput})
		return
	}
	if !s.ingestMu.TryLock() {
		writeJSON(w, http.StatusConflict, errBody{Error: "đang có một lần nạp khác; thử lại sau", Fault: contracts.FaultData})
		return
	}
	defer s.ingestMu.Unlock()
	ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), ingestTimeout)
	defer cancel()
	rep, err := s.app.ingest(ctx)
	if err != nil {
		fault, msg := traces.SanitizeError(err)
		s.log.Error("nạp lại dữ liệu thất bại", "tenant", pr.TenantID, "principal", pr.ID, "fault", fault, "lỗi", msg)
		public := publicMessage(fault, msg)
		if fault == contracts.FaultData { // người gọi đã là vai toàn quyền và xung đột dữ liệu là thứ họ phải xử lý: nói rõ
			public = msg
			if code := pgstore.SQLState(err); code != "" { // lỗi do PG trả: chỉ công bố mã; thông điệp thô (bảng, ràng buộc, giá trị) nằm ở log
				public = "cơ sở dữ liệu từ chối dữ liệu nạp (SQLSTATE " + code + "); chi tiết ở log của server"
			}
		}
		writeJSON(w, contracts.HTTPStatus(err), errBody{Error: public, Fault: publicFault(fault)})
		return
	}
	s.log.Info("đã nạp lại dữ liệu", "tenant", pr.TenantID, "principal", pr.ID, "load_id", rep.LoadID, "observations", rep.Observations,
		"quarantined", rep.Quarantined, "quarantine_new", rep.NewQuarantine, "ms", rep.ElapsedMs)
	writeJSON(w, http.StatusOK, rep)
}

// traceResponse là thân GET /v1/trace/{id} (dạng JSON).
type traceResponse struct {
	Record traces.Record `json:"record"`
	Usage  []usageJSON   `json:"usage"`
}

type usageJSON struct {
	Model        string `json:"model"`
	Calls        int    `json:"calls"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	CacheRead    int    `json:"cache_read"`
	CacheWrite   int    `json:"cache_write"`
}

// traceGet trả vết của một lượt — CHỈ trong tenant của principal (RLS: tenant khác ⇒ không có dòng ⇒ 404, không phân biệt "không
// có" với "của người khác"). Trong cùng tenant, chỉ người đã hỏi lượt đó hoặc người có toàn quyền dữ liệu (fullAccess) đọc được; người
// khác cũng nhận 404. ?format=table trả bảng chữ (cùng bảng với `eais trace`).
func (s *server) traceGet(w http.ResponseWriter, r *http.Request, pr contracts.Principal) {
	deadline(w, 30*time.Second)
	ctx := contracts.WithTenant(r.Context(), pr.TenantID)
	id := r.PathValue("id")
	rec, err := s.st.Decision(ctx, id)
	if err == nil && rec.Decision.Principal != pr.ID && !s.fullAccess(pr) {
		err = pgstore.ErrNotFound
	}
	var usage []pgstore.UsageRecord
	if err == nil {
		usage, err = s.st.Usage(ctx, id)
	}
	if errors.Is(err, pgstore.ErrNotFound) {
		writeJSON(w, http.StatusNotFound, errBody{Error: "không có vết với mã này", Fault: contracts.FaultInput})
		return
	}
	if err != nil {
		fault, msg := traces.SanitizeError(err)
		s.log.Error("đọc vết thất bại", "trace_id", traces.Sanitize(id), "tenant", pr.TenantID, "fault", fault, "lỗi", msg)
		writeJSON(w, contracts.HTTPStatus(err), errBody{Error: publicMessage(fault, msg), Fault: publicFault(fault)})
		return
	}
	if r.URL.Query().Get("format") == "table" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprint(w, traces.FormatTable(rec), formatUsage(usage))
		return
	}
	out := traceResponse{Record: rec, Usage: []usageJSON{}}
	for _, u := range usage {
		out.Usage = append(out.Usage, usageJSON{Model: u.Model, Calls: u.Calls, InputTokens: u.Tokens.InputTokens,
			OutputTokens: u.Tokens.OutputTokens, CacheRead: u.Tokens.CacheRead, CacheWrite: u.Tokens.CacheWrite})
	}
	writeJSON(w, http.StatusOK, out)
}

// formatUsage in usage của lượt dưới bảng vết.
func formatUsage(usage []pgstore.UsageRecord) string {
	var b strings.Builder
	b.WriteString("\nUSAGE MODEL\n")
	if len(usage) == 0 {
		b.WriteString("  (chưa có bản ghi)\n")
	}
	for _, u := range usage {
		fmt.Fprintf(&b, "  %s: %d lời gọi, %d token vào, %d token ra\n", u.Model, u.Calls, u.Tokens.InputTokens, u.Tokens.OutputTokens)
	}
	return b.String()
}
