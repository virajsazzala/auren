package search

import (
	"database/sql"
	"fmt"

	faiss "github.com/DataIntelligenceCrew/go-faiss"
	_ "github.com/mattn/go-sqlite3"
)

type Doc struct {
	FileID  string
	Content string
}

type Index struct {
	idx  faiss.Index
	docs []Doc
}

func getNTotal(idx faiss.Index) (int, bool) {
	type nt1 interface { NTotal() int }
	type nt2 interface { NTotal() int }

	if v, ok := idx.(nt1); ok {
		return int(v.NTotal()), true
	}
	if v, ok := idx.(nt2); ok {
		return int(v.NTotal()), true
	}
	return 0, false
}

func NewIndex(dim int, useIP bool) (*Index, error) {
	var (
		idx faiss.Index
		err error
	)

	if useIP {
		idx, err = faiss.NewIndexFlatIP(dim)
	} else {
		idx, err = faiss.NewIndexFlatL2(dim)
	}

	if err != nil {
		return nil, err
	}

	return &Index{idx: idx, docs: make([]Doc, 0, 1024)}, nil
}

func (i *Index) Add(docs []Doc, vectors [][]float32) error {
	if len(docs) != len(vectors) {
		return fmt.Errorf("mismatched lengths: %d docs, %d vectors", len(docs), len(vectors))
	}

	if len(vectors) == 0 {
		return nil
	}

	dim := len(vectors[0])
	for _, vec := range vectors {
		if len(vec) != dim {
			return fmt.Errorf("vector dimension mismatch: expected %d, got %d", dim, len(vec))
		}
	}

	existing := make(map[string]struct{}, len(i.docs)+len(docs))
	for _, d := range i.docs {
		if d.FileID != "" {
			existing[d.FileID] = struct{}{}
		}
	}

	var (
		filterDocs	[]Doc
		filterVecs	[][]float32
		skipped		[]string
	)
	for idx, d := range docs {
		if d.FileID == "" {
			return fmt.Errorf("empty file ID for doc at index %d", idx)
		}
		if _, ok := existing[d.FileID]; ok {
			skipped = append(skipped, d.FileID)
			continue
		}
		existing[d.FileID] = struct{}{}
		filterDocs = append(filterDocs, d)
		filterVecs = append(filterVecs, vectors[idx])
	}

	if len(filterDocs) == 0 {
		return fmt.Errorf("no new documents to add, all %d were duplicates: %v", len(docs), skipped)
	}

	flatVectors := FlattenVectors(filterVecs)
	if err := i.idx.Add(flatVectors); err != nil {
		return err
	}

	i.docs = append(i.docs, filterDocs...)

	if len(skipped) > 0 {
		return fmt.Errorf("skipped %d duplicate file ids: %v", len(skipped), skipped)
	}

	return nil
}

func (i *Index) Search(query []float32, k int) ([]Result, error) {
	dists, labels, err := i.idx.Search(query, int64(k))
	if err != nil {
		return nil, err
	}

	out := make([]Result, 0, k)
	for j, id := range labels {
		if id >= 0 && int(id) < len(i.docs) {
			out = append(out, Result{
				FileID: i.docs[id].FileID,
				Score:  dists[j],
			})
		}
	}

	return out, nil
}

func (i *Index) Save(path string) error {
	if err := faiss.WriteIndex(i.idx, path+".faiss"); err != nil {
		return fmt.Errorf("failed to save faiss: %w", err)
	}

	db, err := sql.Open("sqlite3", path+".db")
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS docs (
		faiss_id INTEGER PRIMARY KEY,
		file_id TEXT UNIQUE
	);`)
	if err != nil {
		return err
	}

	// replace all mappings
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.Exec("DELETE from docs;"); err != nil {
		return err
	}

	stmt, err := tx.Prepare("INSERT INTO docs(faiss_id, file_id) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for faissID, doc := range i.docs {
		if _, err := stmt.Exec(faissID, doc.FileID); err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (i *Index) Load(path string) error {
	idxImpl, err := faiss.ReadIndex(path+".faiss", 0)
	if err != nil {
		return fmt.Errorf("failed to load faiss: %w", err)
	}
	i.idx = idxImpl

	db, err := sql.Open("sqlite3", path+".db")
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query("SELECT faiss_id, file_id FROM docs")
	if err != nil {
		return err
	}
	defer rows.Close()

	type pair struct {
		id     int
		fileID string
	}

	var pairs []pair
	maxID := -1
	for rows.Next() {
		var fid int
		var fileID string
		if err := rows.Scan(&fid, &fileID); err != nil {
			return err
		}
		pairs = append(pairs, pair{fid, fileID})
		if fid > maxID {
			maxID = fid
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if maxID < 0 {
		i.docs = nil
		return nil
	}

	docs := make([]Doc, maxID+1)
	for _, p := range pairs {
		if docs[p.id].FileID != "" {
			return fmt.Errorf("duplicate mapping for faiss_id %d", p.id)
		}
		docs[p.id] = Doc{FileID: p.fileID}
	}

	// detect missing ids
	for id, d := range docs {
		if d.FileID == "" {
			return fmt.Errorf("missing mapping for faiss_id %d", id)
		}
	}

	if nt, ok := getNTotal(i.idx); ok {
		if nt != len(docs) {
			return fmt.Errorf("faiss index size %d does not match docs size %d", nt, len(docs))
		}
	}

	i.docs = docs
	return nil
}
