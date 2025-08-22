package search

import (
	"encoding/json"
	"os"

	"github.com/virajsazzala/auren/internal/config"
	"github.com/virajsazzala/auren/internal/python"
)

type Result struct {
	FileID  string  `json:"file_id"`
	Score   float32 `json:"score"`
}

var FAISS *Index

func Init() error {
	var err error
	FAISS, err = NewIndex(config.EmbeddingDim, true)
	return err
}

func LoadDocsFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var docs map[string]string
	if err := json.Unmarshal(data, &docs); err != nil {
		return err
	}

	// embed & add
	dlist := make([]Doc, 0, len(docs))
	vecs := make([][]float32, 0, len(docs))
	for fid, content := range docs {
		v, err := python.GetEmbedding(content)
		if err != nil {
			return err
		}
		dlist = append(dlist, Doc{FileID: fid, Content: content})
		vecs = append(vecs, v)
	}

	return FAISS.Add(dlist, vecs)
}

func Query(text string, k int) ([]Result, error) {
	vec, err := python.GetEmbedding(text)
	if err != nil {
		return nil, err
	}

	return FAISS.Search(vec, k)
}

func GetEmbedding(text string) ([]float32, error) {
	return python.GetEmbedding(text)
}