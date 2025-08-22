package search

import (
	"fmt"

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