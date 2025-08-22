package search

import (
	"fmt"
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
	faiss "github.com/DataIntelligenceCrew/go-faiss"
)

type Doc struct {
	FileID string
	Content string
}

type Index struct {
	idx faiss.Index
	docs []Doc
}

// move this to some utils dir later on
func flattenVectors(vectors [][]float32) []float32 {
    if len(vectors) == 0 {
        return nil
    }
    dim := len(vectors[0])
    flat := make([]float32, 0, len(vectors)*dim)
    for _, vec := range vectors {
        flat = append(flat, vec...)
    }
    return flat
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

	return &Index {idx: idx, docs: make([]Doc, 0, 1024)}, nil
}

func (i *Index) Add(docs []Doc, vectors [][]float32) error {
	flatVectors := flattenVectors(vectors)
	if err := i.idx.Add(flatVectors); err != nil {
		return err
	}

	i.docs = append(i.docs, docs...)

	fmt.Println("Added document!")

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
				Score: dists[j],
			})
		}
	}

	return out, nil
}

func (i *Index) Save(path string) error {
	if err := faiss.WriteIndex(i.idx, path + ".faiss"); err != nil {
		return fmt.Errorf("failed to save faiss: %w", err)
	}

	db, err := sql.Open("sqlite3", path + ".db")
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
	tx, _ := db.Begin()
	_, _ = tx.Exec("DELETE from docs;")
	stmt, _ := tx.Prepare("INSERT INTO docs(faiss_id, file_id) VALUES (?, ?)")
	defer stmt.Close()

	for faissID, doc := range i.docs {
		if _, err := stmt.Exec(faissID, doc.FileID); err != nil {
			return err
		}
	}
	tx.Commit()

	return nil
}


func (i *Index) Load(path string) error {
	idxImpl, err := faiss.ReadIndex(path + ".faiss", 0)
	if err != nil {
		return fmt.Errorf("failed to load faiss: %w", err)
	}
	i.idx = idxImpl

	db, err := sql.Open("sqlite3", path + ".db")
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query("SELECT faiss_id, file_id FROM docs ORDER BY faiss_id ASC")
	if err != nil {
		return err
	}
	defer rows.Close()

	var docs []Doc
	for rows.Next() {
		var fid int
		var fileID string
		if err := rows.Scan(&fid, &fileID); err != nil {
			return err
		}
		docs = append(docs, Doc{FileID: fileID})
	}
	i.docs = docs
	return nil
}


