
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
	"dictionary-translator/backend/internal/fileprocessor"
)

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("Write JSON response: %v", err)
	}
}

func main() {
	// --------------------------------------------------
	// 1. Load the Hebrew letter concordance
	// --------------------------------------------------

	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "../data"
	}

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

	// --------------------------------------------------
	// 2. Load the Hebrew alphabet chart
	// --------------------------------------------------

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

	// Map character representations to their letter names.
	characterIndex := alphabet.BuildIndex(chart)

	// --------------------------------------------------
	// 3. Define the translation function
	// --------------------------------------------------

	translateText := func(input string) (string, error) {
		letterNames, err := alphabet.ResolveSequence(
			characterIndex,
			input,
		)
		if err != nil {
			return "", err
		}

		meanings := make([]string, 0, len(letterNames))

		for _, letterName := range letterNames {
			if letterName == alphabet.UnknownCharacter {
				meanings = append(
					meanings,
					alphabet.UnknownCharacter,
				)
				continue
			}

			entry, found := concordance.LookupLetter(
				letterIndex,
				letterName,
			)

			if !found || strings.TrimSpace(entry.PreferredMeaning) == "" {
				meanings = append(
					meanings,
					alphabet.UnknownCharacter,
				)
				continue
			}

			meanings = append(meanings, entry.PreferredMeaning)
		}

		return strings.Join(meanings, " "), nil
	}

	// --------------------------------------------------
	// 4. Process the input text file on application startup
	// --------------------------------------------------

	inputDir := os.Getenv("FILES_INPUT_DIR")
	if inputDir == "" {
		inputDir = "../files/input"
	}

	outputDir := os.Getenv("FILES_OUTPUT_DIR")
	if outputDir == "" {
		outputDir = "../files/output"
	}

	inputPath := filepath.Join(inputDir, "example.txt")

	outputPath, err := fileprocessor.ProcessFile(
		inputPath,
		outputDir,
		translateText,
	)

	if err != nil {
		// Keep the API running even if the input file is missing.
		log.Printf("File processing skipped or failed: %v", err)
	} else {
		log.Printf("Output saved to: %s", outputPath)
	}

	// --------------------------------------------------
	// 5. Register API routes
	// --------------------------------------------------

	mux := http.NewServeMux()

	// Health check.
	mux.HandleFunc("GET /api/v1/health", func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	// Translate a sequence through the concordance.
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

		letterNames, err := alphabet.ResolveSequence(
			characterIndex,
			input,
		)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": err.Error(),
			})
			return
		}

		letters := make([]string, 0, len(letterNames))
		meanings := make([]string, 0, len(letterNames))

		for _, letterName := range letterNames {
			if letterName == alphabet.UnknownCharacter {
				letters = append(
					letters,
					alphabet.UnknownCharacter,
				)
				meanings = append(
					meanings,
					alphabet.UnknownCharacter,
				)
				continue
			}

			entry, found := concordance.LookupLetter(
				letterIndex,
				letterName,
			)

			if !found || strings.TrimSpace(entry.PreferredMeaning) == "" {
				letters = append(letters, letterName)
				meanings = append(
					meanings,
					alphabet.UnknownCharacter,
				)
				continue
			}

			letters = append(letters, letterName)
			meanings = append(meanings, entry.PreferredMeaning)
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"input":    input,
			"letters":  letters,
			"meanings": meanings,
			"result":   strings.Join(meanings, " "),
		})
	})

	// --------------------------------------------------
	// 6. Start the API server
	// --------------------------------------------------

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Concordance API listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
