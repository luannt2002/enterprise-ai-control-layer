package ingestion

import (
	"path/filepath"

	"github.com/luannt2002/enterprise-ai-control-layer/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-control-layer/data/sourceoftruth"
)

// LoadDirectory trả danh mục thực thể của tenant cho LUỒNG HỎI: bí danh chung + mọi mã theo nguồn
// (người dùng có thể hỏi "asset-8392" hay "EX003" như hỏi "máy 03").
func LoadDirectory(dir string, sm SourceMap) (entityresolution.MemoryDirectory, error) {
	ix, err := loadIndex(filepath.Join(dir, sm.EntitiesFile), sm.EntityLists)
	if err != nil {
		return nil, err
	}
	out := make(entityresolution.MemoryDirectory, 0, len(ix.base))
	for _, e := range ix.base {
		if codes := ix.codes[e.ID]; codes != nil {
			aliases := append([]entityresolution.Alias(nil), e.Aliases...)
			for ns, code := range codes {
				aliases = append(aliases, entityresolution.Alias{Text: code, Source: ns})
			}
			e.Aliases = aliases
		}
		out = append(out, e)
	}
	return out, nil
}

// Facts đổi các quan sát thành fact cho nguồn sự thật — khoá bản ghi nguồn đi theo làm id để
// mọi con số truy được về đúng dòng dữ liệu gốc.
func (r Result) Facts() sourceoftruth.MemoryStore {
	out := make(sourceoftruth.MemoryStore, 0, len(r.Observations))
	for _, o := range r.Observations {
		out = append(out, sourceoftruth.Fact{
			ID: o.Key, Metric: o.Metric, EntityID: o.Entity, OccurredOn: o.On, Value: o.Value,
			Source: sourceoftruth.Observation{ID: o.Key, SourceType: o.Source, SourceRef: o.Key, Raw: o.Raw, ObservedAt: o.SentAt, ReportedBy: o.Sender, Note: o.Note},
		})
	}
	return out
}
