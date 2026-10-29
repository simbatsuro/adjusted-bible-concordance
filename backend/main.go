
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
	"dictionary-translator/backend/internal/phraseconcordance"
)

func main() {
	dataDir := getEnv("DATA_DIR", "../data")
	inputDir := getEnv("FILES_INPUT_DIR", "../files/input")
	outputDir := getEnv("FILES_OUTPUT_DIR", "../files/output")
	port := getEnv("PORT", "8080")

	// Load the letter concordance.
	var letterData concordance.Concordance
	if err := loadJSON(
		filepath.Join(dataDir, "hebrew-letter-concordance.json"),
		&letterData,
	); err != nil {
		log.Fatalf("load letter concordance: %v", err)
	}

	letterIndex := concordance.BuildLetterIndex(letterData)

	// Load and validate the alphabet chart.
	var chart alphabet.Chart
	if err := loadJSON(
		filepath.Join(dataDir, "hebrew-alphabet-chart.json"),
		&chart,
	); err != nil {
		log.Fatalf("load alphabet chart: %v", err)
	}

	if err := chart.Validate(); err != nil {
		log.Fatalf("validate alphabet chart: %v", err)
	}

	characterIndex := alphabet.BuildIndex(chart)

	// Translate input using the letter concordance.
	translateLetter := func(input string) (string, error) {
		letterNames, err := alphabet.ResolveSequence(characterIndex, input)
		if err != nil {
			return alphabet.UnknownCharacter, nil
		}

		meanings := make([]string, 0, len(letterNames))

		for _, letterName := range letterNames {
			if letterName == alphabet.UnknownCharacter {
				meanings = append(meanings, alphabet.UnknownCharacter)
				continue
			}

			entry, found := concordance.LookupLetter(letterIndex, letterName)
			if !found || strings.TrimSpace(entry.PreferredMeaning) == "" {
				meanings = append(meanings, alphabet.UnknownCharacter)
				continue
			}

			meanings = append(meanings, entry.PreferredMeaning)
		}

		if len(meanings) == 0 {
			return alphabet.UnknownCharacter, nil
		}

		return strings.Join(meanings, " "), nil
	}

	// Load the phrase concordance.
	phraseData, err := phraseconcordance.Load(
		filepath.Join(dataDir, "hebrew-phrase-concordance.json"),
	)
	if err != nil {
		log.Fatalf("load phrase concordance: %v", err)
	}

	phraseIndex := phraseconcordance.BuildIndex(phraseData)

	// Load prefixes and suffixes.
	prefixes, err := phraseconcordance.LoadAffixes(
		filepath.Join(dataDir, "prefixes.json"),
		"prefixes",
	)
	if err != nil {
		log.Fatalf("load prefixes: %v", err)
	}

	suffixes, err := phraseconcordance.LoadAffixes(
		filepath.Join(dataDir, "suffixes.json"),
		"suffixes",
	)
	if err != nil {
		log.Fatalf("load suffixes: %v", err)
	}

	prefixIndex := phraseconcordance.BuildAffixIndex(prefixes)
	suffixIndex := phraseconcordance.BuildAffixIndex(suffixes)

	// Translate input using exact phrase lookup first, followed by
	// prefix/suffix analysis when an exact match is unavailable.
	translatePhrase := func(input string) (string, error) {
		return phraseconcordance.LookupWithAffixes(
			phraseIndex,
			prefixIndex,
			suffixIndex,
			input,
		), nil
	}

	// Generate separate letter and phrase output files.
	inputPath := filepath.Join(inputDir, "example.txt")

	letterOutput, err := fileprocessor.ProcessLetterFile(
		inputPath,
		outputDir,
		translateLetter,
	)
	if err != nil {
		log.Fatalf("process letter output: %v", err)
	}

	phraseOutput, err := fileprocessor.ProcessPhraseFile(
		inputPath,
		outputDir,
		translatePhrase,
	)
	if err != nil {
		log.Fatalf("process phrase output: %v", err)
	}

	log.Printf("Letter output created: %s", letterOutput)
	log.Printf("Phrase output created: %s", phraseOutput)

	// Health endpoint.
	http.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "ok",
		})
	})

	// Letter concordance endpoint.
	http.HandleFunc("/api/v1/concordance/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		input := strings.TrimPrefix(r.URL.Path, "/api/v1/concordance/")
		if strings.TrimSpace(input) == "" {
			http.Error(w, "missing letter name", http.StatusBadRequest)
			return
		}

		entry, found := concordance.LookupLetter(letterIndex, input)
		if !found {
			writeJSON(w, http.StatusNotFound, map[string]string{
				"translation": alphabet.UnknownCharacter,
			})
			return
		}

		writeJSON(w, http.StatusOK, entry)
	})

	// Phrase endpoint, including prefix/suffix fallback.
	http.HandleFunc("/api/v1/phrase/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		input := strings.TrimPrefix(r.URL.Path, "/api/v1/phrase/")
		if strings.TrimSpace(input) == "" {
			http.Error(w, "missing phrase", http.StatusBadRequest)
			return
		}

		translation := phraseconcordance.LookupWithAffixes(
			phraseIndex,
			prefixIndex,
			suffixIndex,
			input,
		)

		writeJSON(w, http.StatusOK, map[string]string{
			"input":       input,
			"translation": translation,
		})
	})

	log.Printf("Dictionary translator listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func loadJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, target); err != nil {
		return err
	}

	return nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	return value
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write JSON response: %v", err)
	}
}
