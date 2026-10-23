package reports

import (
	"context"
	"sort"
	"sync"
)

// Memory là kho bộ nhớ cho test. Chạy cùng engine với kho PG; khác ở chỗ tuần tự hoá bằng một khoá toàn kho (đủ cho một tiến trình),
// nên không cần khoá tư vấn.
type Memory struct {
	mu   sync.Mutex
	opt  Options
	data map[string]*memTenant
}

type memTenant struct {
	recs     map[string]*Record
	versions map[string][]versionRow
	fps      map[string][]string
	history  map[string][]HistoryEntry
	ledger   []DeliveryEntry
}

// NewMemory dựng kho bộ nhớ.
func NewMemory(opt Options) *Memory { return &Memory{opt: opt, data: map[string]*memTenant{}} }

var _ Store = (*Memory)(nil)

func (m *Memory) tenantData(id string) *memTenant {
	t := m.data[id]
	if t == nil {
		t = &memTenant{recs: map[string]*Record{}, versions: map[string][]versionRow{}, fps: map[string][]string{}, history: map[string][]HistoryEntry{}}
		m.data[id] = t
	}
	return t
}

// run chạy fn như một giao dịch: chỉ khi fn xong (hoặc lỗi mang audit) thì mới ghi vào kho.
func (m *Memory) run(ctx context.Context, fn func(c *opctx) error) error {
	tenant, err := tenantFrom(ctx)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	c := newOpctx(tenant, m.opt, &memTx{t: m.tenantData(tenant)})
	_, err = c.close(fn(c))
	return err
}

func (m *Memory) read(ctx context.Context, fn func(t *memTenant) error) error {
	tenant, err := tenantFrom(ctx)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return fn(m.tenantData(tenant))
}

func (m *Memory) Deliver(ctx context.Context, batchID string, ordinal int, it Item) (DeliveryEntry, error) {
	var out DeliveryEntry
	err := m.run(ctx, func(c *opctx) error {
		e, err := c.deliver(batchID, ordinal, it)
		out = e
		return err
	})
	return out, err
}

func (m *Memory) DeliverBatch(ctx context.Context, items []Item) (Receipt, error) {
	return deliverBatch(ctx, m, items)
}

func (m *Memory) Decide(ctx context.Context, cmd Command) (Record, error) {
	var out Record
	err := m.run(ctx, func(c *opctx) error {
		if err := c.decide(cmd); err != nil {
			return err
		}
		out = c.cache[cmd.Key].clone()
		return nil
	})
	return out, err
}

func (m *Memory) Retract(ctx context.Context, key, actor, why string) error {
	return m.run(ctx, func(c *opctx) error { return c.retract(key, actor, why) })
}

func (m *Memory) Amend(ctx context.Context, key string, it Item, actor, why string) error {
	return m.run(ctx, func(c *opctx) error { return c.amend(key, it, actor, why) })
}

func (m *Memory) Record(ctx context.Context, key string) (Record, bool, error) {
	var out Record
	var ok bool
	err := m.read(ctx, func(t *memTenant) error {
		if r, found := t.recs[key]; found {
			out, ok = r.clone(), true
		}
		return nil
	})
	return out, ok, err
}

func (m *Memory) Records(ctx context.Context) ([]Record, error) {
	var out []Record
	err := m.read(ctx, func(t *memTenant) error {
		for _, r := range t.recs {
			out = append(out, r.clone())
		}
		return nil
	})
	sortRecords(out)
	return out, err
}

func (m *Memory) Ledger(ctx context.Context, batchID string) ([]DeliveryEntry, error) {
	var out []DeliveryEntry
	err := m.read(ctx, func(t *memTenant) error {
		for _, e := range t.ledger {
			if batchID == "" || e.BatchID == batchID {
				out = append(out, e)
			}
		}
		return nil
	})
	sortLedger(out)
	return out, err
}

func (m *Memory) History(ctx context.Context, key string) ([]HistoryEntry, error) {
	var out []HistoryEntry
	err := m.read(ctx, func(t *memTenant) error {
		out = append(out, t.history[key]...)
		return nil
	})
	return out, err
}

func (m *Memory) Receipt(ctx context.Context, batchID string) (Receipt, error) {
	es, err := m.Ledger(ctx, batchID)
	return BuildReceipt(batchID, es), err
}

func (m *Memory) Totals(ctx context.Context) ([]Total, error) {
	rs, err := m.Records(ctx)
	if err != nil {
		return nil, err
	}
	return totalsOf(rs), nil
}

// memTx cài txn trên một tenant của kho bộ nhớ. Đọc trả bản chép; ghi chép vào kho: engine không giữ con trỏ vào dữ liệu của kho.
type memTx struct{ t *memTenant }

func (x *memTx) getRecord(key string) (*Record, error) {
	if r, ok := x.t.recs[key]; ok {
		c := r.clone()
		return &c, nil
	}
	return nil, nil
}

func (x *memTx) versionsOf(key string) ([]versionRow, error) {
	return append([]versionRow(nil), x.t.versions[key]...), nil
}

func (x *memTx) findByFingerprint(fp string) (string, error) {
	keys := x.t.fps[fp]
	if len(keys) == 0 {
		return "", nil
	}
	best := keys[0]
	for _, k := range keys[1:] {
		if k < best {
			best = k
		}
	}
	return best, nil
}

func (x *memTx) slotRecords(entity, day string) ([]*Record, error) {
	var out []*Record
	for _, r := range x.t.recs {
		if r.Entity == entity && r.OccurredOn == day {
			c := r.clone()
			out = append(out, &c)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key < out[j].Key })
	return out, nil
}

func (x *memTx) insertRecord(r *Record) error {
	if _, ok := x.t.recs[r.Key]; ok {
		return errKeyRace
	}
	c := r.clone()
	x.t.recs[r.Key] = &c
	return nil
}

func (x *memTx) updateRecord(r *Record) error {
	c := r.clone()
	x.t.recs[r.Key] = &c
	return nil
}

func (x *memTx) insertVersion(v versionRow) error {
	x.t.versions[v.Key] = append(x.t.versions[v.Key], v)
	for _, k := range x.t.fps[v.Fingerprint] {
		if k == v.Key {
			return nil
		}
	}
	x.t.fps[v.Fingerprint] = append(x.t.fps[v.Fingerprint], v.Key)
	return nil
}

func (x *memTx) insertHistory(h HistoryEntry) error {
	h.Seq = len(x.t.history[h.Key]) + 1
	x.t.history[h.Key] = append(x.t.history[h.Key], h)
	return nil
}

func (x *memTx) insertLedger(e DeliveryEntry) error {
	x.t.ledger = append(x.t.ledger, e)
	return nil
}
