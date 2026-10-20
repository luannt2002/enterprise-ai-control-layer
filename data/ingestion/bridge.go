package ingestion

import (
	"path/filepath"

	"github.com/luannt2002/enterprise-ai-systems/data/entityresolution"
	"github.com/luannt2002/enterprise-ai-systems/data/sourceoftruth"
)

// LoadDirectory trả danh mục thực thể của tenant cho LUỒNG HỎI: bí danh chung + mọi mã theo nguồn
// (người dùng có thể hỏi "asset-8392" hay "EX003" như hỏi "máy 03").
func LoadDirectory(dir string, sm SourceMap) (entityresolution.MemoryDirectory, error) {
	ix, err := loadIndex(filepath.Join(dir, sm.EntitiesFile))
	if err != nil {
		return nil, err
	}
	out := make(entityresolution.MemoryDirectory, 0, len(ix.base))
	for _, e := range ix.base {
		for _, q := range ix.raw.Equipment {
			if q.ID != e.ID {
				continue
			}
			aliases := append([]entityresolution.Alias(nil), e.Aliases...)
			for ns, code := range q.SourceIDs {
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
			Source: sourceoftruth.Observation{ID: o.Key, SourceType: o.Source, SourceRef: o.Key, Raw: o.Raw},
		})
	}
	return out
}
