package python

import (
	"fmt"
	"bytes"
	"net/http"
	"encoding/json"

	"github.com/virajsazzala/auren/internal/config"
)

func GetEmbedding(text string) ([]float32, error) {
    reqBody, _ := json.Marshal(map[string]string{"text": text})

    resp, err := http.Post(fmt.Sprintf("http://127.0.0.1%s/%s", config.EmbedServerAddr, config.EmbedEndpoint), "application/json", bytes.NewBuffer(reqBody))
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var vec []float32
    if err := json.NewDecoder(resp.Body).Decode(&vec); err != nil {
        return nil, err
    }
    return vec, nil
}
