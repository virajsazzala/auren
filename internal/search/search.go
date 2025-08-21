package search

import (
	"encoding/json"
	"os"

	"github.com/virajsazzala/auren/internal/config"
	"github.com/virajsazzala/auren/internal/python"
)

type Result struct {
	Document string  `json:"document"`
	Score    float32 `json:"score"`
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

	var docs []string
	if err := json.Unmarshal(data, &docs); err != nil {
		return err
	}

	// embed & add
	vecs := make([][]float32, 0, len(docs))
	for _, d := range docs {
		v, err := python.GetEmbedding(d)
		if err != nil {
			return err
		}
		vecs = append(vecs, v)
	}

	return FAISS.Add(docs, vecs)
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