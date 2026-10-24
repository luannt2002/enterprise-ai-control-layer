package pgstore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/luannt2002/enterprise-ai-control-layer/core/contracts"
	"github.com/luannt2002/enterprise-ai-control-layer/intelligence/rag"
)

// EmbedKnowledge đồng bộ vector của kho tri thức tenant với không gian vector hiện tại (Semantic.Embedder.Name(), migration 0008):
// nhúng chữ của đoạn chưa có vector — chữ đó được GỬI RA embedder đã cấu hình; chỉ đoạn của bản đã duyệt (rag EmbedTexts, F-017) — và
// xoá vector của chữ không còn trong kho (kể cả vector cũ của bản chưa duyệt). Chạy lại
// được: chữ không đổi thì không nhúng lại. Gọi sau mỗi lần nạp tài liệu; chưa gọi thì lượt hỏi báo thiếu vector (rag.Semantic).
func (s *Store) EmbedKnowledge(ctx context.Context, tenant contracts.TenantID) (added, removed int, err error) {
	if s.Semantic == nil || s.Semantic.Embedder == nil {
		return 0, 0, contracts.Errorf(contracts.FaultCode, "chưa cấu hình embedding (Store.Semantic)")
	}
	space := s.Semantic.Embedder.Name()
	ix, err := s.knowledgeIndex(ctx, tenant)
	if err != nil {
		return 0, 0, err
	}
	have := map[string]bool{}
	if err := s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, "SELECT text_hash FROM document_vectors WHERE space = $1", space)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var h string
			if err := rows.Scan(&h); err != nil {
				return err
			}
			have[h] = true
		}
		return rows.Err()
	}); err != nil {
		return 0, 0, err
	}
	var todo []string
	for _, t := range ix.EmbedTexts() {
		if !have[rag.EmbedKey(t)] {
			todo = append(todo, t)
		}
	}
	var vecs [][]float32
	if len(todo) > 0 {
		// Nhúng NGOÀI giao dịch: lời gọi mạng có thể chờ hạn mức hàng phút, không giữ kết nối PG trong lúc đó.
		if vecs, err = s.Semantic.Embedder.Embed(ctx, todo, false); err != nil {
			return 0, 0, err
		}
		if len(vecs) != len(todo) {
			return 0, 0, contracts.Errorf(contracts.FaultCode, fmt.Sprintf("embedder trả %d vector cho %d chữ", len(vecs), len(todo)))
		}
	}
	err = s.inTenant(ctx, tenant, func(tx pgx.Tx) error {
		// Cùng khoá với Load, và tập vector cần GIỮ tính lại từ kho HIỆN TẠI đọc dưới khoá: lần nhúng chạy trên kho cũ (Load mới xen
		// vào lúc đang chờ embedder) không được xoá vector của kho mới (soát độc lập 23/10).
		if _, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock($1)", loadLockKey(tenant)); err != nil {
			return err
		}
		docs, err := readDocuments(ctx, tx, tenant, nil)
		if err != nil {
			return err
		}
		keep := []string{} // không để nil: pgx gửi nil thành NULL, ANY(NULL) không xoá dòng nào (kho rỗng giữ rác — soát độc lập 23/10)
		for _, t := range rag.NewMemoryIndex(string(tenant), docs).EmbedTexts() {
			keep = append(keep, rag.EmbedKey(t))
		}
		b := &pgx.Batch{}
		for i, t := range todo {
			b.Queue(`INSERT INTO document_vectors (tenant_id, space, text_hash, vec) VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`,
				string(tenant), space, rag.EmbedKey(t), vecs[i])
		}
		if err := tx.SendBatch(ctx, b).Close(); err != nil {
			return err
		}
		tag, err := tx.Exec(ctx, "DELETE FROM document_vectors WHERE space = $1 AND NOT (text_hash = ANY($2))", space, keep)
		removed = int(tag.RowsAffected())
		return err
	})
	if err != nil {
		return 0, 0, err
	}
	return len(todo), removed, nil
}

// loadVectors đọc vector của không gian space trong giao dịch của tenant; mark là dấu của các dòng đã đọc (như dấu nạp tài liệu).
func loadVectors(ctx context.Context, tx pgx.Tx, space string) (map[string][]float32, []string, error) {
	rows, err := tx.Query(ctx, "SELECT text_hash, vec, xmin::text FROM document_vectors WHERE space = $1", space)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	out := map[string][]float32{}
	var xmins []string
	for rows.Next() {
		var h, x string
		var v []float32
		if err := rows.Scan(&h, &v, &x); err != nil {
			return nil, nil, err
		}
		out[h] = v
		xmins = append(xmins, x)
	}
	return out, xmins, rows.Err()
}
