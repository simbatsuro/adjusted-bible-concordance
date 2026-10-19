
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"dictionary-translator/backend/internal/dictionary"
)

func main() {
	dataPath := os.Getenv("DATA_DIR")
	if dataPath == "" {
		dataPath = "../../data"
	}

	dict, err := dictionary.Load(
		dataPath + "/languages/en/dictionary.json",
	)
	if err != nil {
		log.Fatalf("Load dictionary: %v", err)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	mux.HandleFunc("GET /api/v1/dictionary/{word}", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		word := strings.TrimSpace(r.PathValue("word"))
		if word == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "word is required",
			})
			return
		}

		entry, found := dict.Lookup(word)
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"error": "word not found",
			})
			return
		}

		writeJSON(w, http.StatusOK, entry)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Dictionary API listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("Write JSON response: %v", err)
	}
}
