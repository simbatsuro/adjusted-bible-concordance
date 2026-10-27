
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"dictionary-translator/backend/internal/alphabet"
	"dictionary-translator/backend/internal/concordance"
)

type LookupResponse struct {
	Letter      string                    `json:"letter"`
	Alphabet    alphabet.Entry             `json:"alphabet"`
	Concordance concordance.LetterEntry    `json:"concordance"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("Write JSON response: %v", err)
	}
}

func main() {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "../data"
	}

	// Load the Hebrew letter concordance.
	concordancePath := filepath.Join(
		dataDir,
		"hebrew-letter-concordance.json",
	)

	concordanceJSON, err := os.ReadFile(concordancePath)
	if err != nil {
		log.Fatalf("Read concordance: %v", err)
	}

	var concordanceData concordance.Concordance
	if err := json.Unmarshal(concordanceJSON, &concordanceData); err != nil {
		log.Fatalf("Parse concordance JSON: %v", err)
	}

	letterIndex := concordance.BuildLetterIndex(concordanceData)

	// Load the Hebrew alphabet chart.
	chartPath := filepath.Join(
		dataDir,
		"hebrew-alphabet-chart.json",
	)

	chartJSON, err := os.ReadFile(chartPath)
	if err != nil {
		log.Fatalf("Read alphabet chart: %v", err)
	}

	var chart alphabet.Chart
	if err := json.Unmarshal(chartJSON, &chart); err != nil {
		log.Fatalf("Parse alphabet chart JSON: %v", err)
	}

	if err := chart.Validate(); err != nil {
		log.Fatalf("Validate alphabet chart: %v", err)
	}

	// Map all character representations to their letter names.
	characterIndex := alphabet.BuildIndex(chart)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	})


	mux.HandleFunc("GET /api/v1/concordance/{input}", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		input := strings.TrimSpace(r.PathValue("input"))
	
		if input == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "letter name or character sequence is required",
			})
			return
		}
	
		// Resolve the input into individual letter names.
		letterNames, err := alphabet.ResolveSequence(characterIndex, input)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
			return
		}
	
		var meanings []string
		var letters []string

		for _, letterName := range letterNames {
			if letterName == alphabet.UnknownCharacter {
				letters = append(letters, alphabet.UnknownCharacter)
				meanings = append(meanings, alphabet.UnknownCharacter)
				continue
			}

			entry, found := concordance.LookupLetter(letterIndex, letterName)
			if !found || strings.TrimSpace(entry.PreferredMeaning) == "" {
				letters = append(letters, letterName)
				meanings = append(meanings, alphabet.UnknownCharacter)
				continue
			}

			letters = append(letters, letterName)
			meanings = append(meanings, entry.PreferredMeaning)
		}

	
		// for _, letterName := range letterNames {
		// 	entry, found := concordance.LookupLetter(letterIndex, letterName)
		// 	if !found {
		// 		writeJSON(w, http.StatusNotFound, map[string]string{
		// 			"error": "concordance entry not found for " + letterName,
		// 		})
		// 		return
		// 	}
	
		// 	if strings.TrimSpace(entry.PreferredMeaning) == "" {
		// 		writeJSON(w, http.StatusInternalServerError, map[string]string{
		// 			"error": "preferred meaning is missing for " + letterName,
		// 		})
		// 		return
		// 	}
	
		// 	letters = append(letters, letterName)
		// 	meanings = append(meanings, entry.PreferredMeaning)
		// }
	
		writeJSON(w, http.StatusOK, map[string]any{
			"input":    input,
			"letters":  letters,
			"meanings": meanings,
			"result":   strings.Join(meanings, " "),
		})
	})
	

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Concordance API listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
