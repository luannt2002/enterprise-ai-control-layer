package gen

// Hai kiểu mục của manifest.json (CONTRACT §3): quan sát kỳ vọng hoặc cách ly kèm lý do.
type obsEntry struct {
	Tenant   string `json:"tenant"`
	Key      string `json:"key"`
	WorldID  string `json:"world_id,omitempty"` // bản ghi nền không có id world
	Expect   string `json:"expect"`
	Entity   string `json:"entity"`
	Semantic string `json:"semantic"`
	Source   string `json:"source"`
	On       string `json:"on"`
	ValueL   string `json:"value_l"`
	Approx   bool   `json:"approx"`
}

type quarEntry struct {
	Tenant  string `json:"tenant"`
	Key     string `json:"key"`
	WorldID string `json:"world_id,omitempty"`
	Expect  string `json:"expect"`
	Reason  string `json:"reason"`
}

// entryFor dựng mục manifest của bản ghi world: quan sát chép từ world, cách ly mang reason của world.
func entryFor(tenant, key string, r record) any {
	if r.quarantined() {
		return quarEntry{Tenant: tenant, Key: key, WorldID: r.ID, Expect: "quarantine", Reason: r.Reason}
	}
	return obsEntry{Tenant: tenant, Key: key, WorldID: r.ID, Expect: "observation", Entity: r.Entity,
		Semantic: r.Semantic, Source: r.Source, On: r.On, ValueL: r.ValueL, Approx: r.Approx}
}

// baseEntry dựng mục manifest của bản ghi nền: reason rỗng là quan sát sạch, ngược lại là cách ly.
func baseEntry(tenant, file, id, entity, semantic, source, on, valueL, reason string) any {
	key := file + "#" + id
	if reason != "" {
		return quarEntry{Tenant: tenant, Key: key, Expect: "quarantine", Reason: reason}
	}
	return obsEntry{Tenant: tenant, Key: key, Expect: "observation", Entity: entity,
		Semantic: semantic, Source: source, On: on, ValueL: valueL}
}
