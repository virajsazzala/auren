package main

import (
	"log"
	"net/http"

	"github.com/virajsazzala/auren/internal/api"
	"github.com/virajsazzala/auren/internal/search"
	"github.com/virajsazzala/auren/internal/config"
)

func main() {
	if err := search.Init(); err != nil {
		log.Fatal("FAISS init error: ", err)
	}

	if err := search.LoadDocsFromFile("data/docs.json"); err != nil {
		log.Println("warn: failed loading sample docs: ", err)
	}

	http.HandleFunc("/search", api.SearchHandler)
	http.HandleFunc("/add", api.AddHandler)
	http.HandleFunc("/save", api.SaveHandler)
	http.HandleFunc("/load", api.LoadHandler)

	log.Println("server on", config.ServerAddr)
	log.Fatal(http.ListenAndServe(config.ServerAddr, nil))
}