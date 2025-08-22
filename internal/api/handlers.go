package api

import (
	"encoding/json"
	"net/http"

	"github.com/virajsazzala/auren/internal/search"
	"github.com/virajsazzala/auren/internal/config"
)

type SearchRequest struct {
	Query string `json:"query"`
	TopK  *int   `json:"top_k,omitempty"`
}

type AddRequest struct {
	Documents map[string]string `json:"documents"`
}

func SearchHandler(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Query == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	k := config.TopK
	if req.TopK != nil && *req.TopK > 0 {
		k = *req.TopK
	}

	res, err := search.Query(req.Query, k)
	if err != nil {
		http.Error(w, "search error: " + err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(res)
}

func AddHandler(w http.ResponseWriter, r *http.Request) {
	var req AddRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Documents) == 0 {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	dlist := make([]search.Doc, 0, len(req.Documents))
	vecs := make([][]float32, 0, len(req.Documents))

	for fid, content := range req.Documents {
		v, err := search.GetEmbedding(content) // small proxy
		if err != nil {
			http.Error(w, "embedding error: " + err.Error(), http.StatusInternalServerError)
			return
		}
		dlist = append(dlist, search.Doc{FileID: fid, Content: content})
		vecs = append(vecs, v)
	}

	if err := search.FAISS.Add(dlist, vecs); err != nil {
		http.Error(w, "faiss add error: " + err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"status":"ok","added" : len(req.Documents)})
}

func SaveHandler(w http.ResponseWriter, r *http.Request) {
	if err := search.SaveIndex(config.IndexPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "saved"})
}

func LoadHandler(w http.ResponseWriter, r *http.Request) {
	if err := search.LoadIndex(config.IndexPath); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "loaded"})
}