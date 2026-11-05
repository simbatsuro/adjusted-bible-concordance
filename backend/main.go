package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"dictionary-translator/backend/internal/alphabet"
	"dictionary-translator/backend/internal/concordance"
	"dictionary-translator/backend/internal/fileprocessor"
	"dictionary-translator/backend/internal/phraseconcordance"
)

const maxRequestSize = 10 << 20 // 10 MiB

var tokenPattern = regexp.MustCompile(`\S+|\s+`)

type processRequest struct {
	Mode string `json:"mode"`
	Text string `json:"text"`
}

func main() {
	dataDir := getEnv("DATA_DIR", "../data")
	inputDir := getEnv("FILES_INPUT_DIR", "../files/input")
	outputDir := getEnv("FILES_OUTPUT_DIR", "../files/output")
	port := getEnv("PORT", "8080")

	var letterData concordance.Concordance
	if err := loadJSON(filepath.Join(dataDir, "hebrew-letter-concordance.json"), &letterData); err != nil {
		log.Fatalf("load letter concordance: %v", err)
	}
	letterIndex := concordance.BuildLetterIndex(letterData)

	var chart alphabet.Chart
	if err := loadJSON(filepath.Join(dataDir, "hebrew-alphabet-chart.json"), &chart); err != nil {
		log.Fatalf("load alphabet chart: %v", err)
	}
	if err := chart.Validate(); err != nil {
		log.Fatalf("validate alphabet chart: %v", err)
	}
	characterIndex := alphabet.BuildIndex(chart)

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

	phraseData, err := phraseconcordance.Load(
		filepath.Join(dataDir, "hebrew-phrase-concordance-HK-mapped.json"),
	)
	if err != nil {
		log.Fatalf("load phrase concordance: %v", err)
	}
	phraseIndex := phraseconcordance.BuildIndex(phraseData)

	prefixes, err := phraseconcordance.LoadAffixes(
		filepath.Join(dataDir, "prefixes.json"), "prefixes",
	)
	if err != nil {
		log.Fatalf("load prefixes: %v", err)
	}
	suffixes, err := phraseconcordance.LoadAffixes(
		filepath.Join(dataDir, "suffixes.json"), "suffixes",
	)
	if err != nil {
		log.Fatalf("load suffixes: %v", err)
	}
	prefixIndex := phraseconcordance.BuildAffixIndex(prefixes)
	suffixIndex := phraseconcordance.BuildAffixIndex(suffixes)

	translatePhrase := func(input string) (string, error) {
		return phraseconcordance.LookupWithAffixes(
			phraseIndex, prefixIndex, suffixIndex, input,
		), nil
	}

	inputPath := filepath.Join(inputDir, "example.txt")
	letterOutput, err := fileprocessor.ProcessLetterFile(
		inputPath, outputDir, translateLetter,
	)
	if err != nil {
		log.Fatalf("process letter output: %v", err)
	}
	phraseOutput, err := fileprocessor.ProcessPhraseFile(
		inputPath, outputDir, translatePhrase,
	)
	if err != nil {
		log.Fatalf("process phrase output: %v", err)
	}
	log.Printf("Letter output created: %s", letterOutput)
	log.Printf("Phrase output created: %s", phraseOutput)

	http.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

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
			phraseIndex, prefixIndex, suffixIndex, input,
		)
		writeJSON(w, http.StatusOK, map[string]string{
			"input":       input,
			"translation": translation,
		})
	})

	http.HandleFunc("/api/v1/process", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
		defer r.Body.Close()

		var request processRequest
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&request); err != nil {
			http.Error(w, "invalid request JSON or request too large", http.StatusBadRequest)
			return
		}

		var extra any
		if err := decoder.Decode(&extra); err != io.EOF {
			http.Error(w, "request must contain one JSON object", http.StatusBadRequest)
			return
		}

		if strings.TrimSpace(request.Text) == "" {
			http.Error(w, "text is required", http.StatusBadRequest)
			return
		}
		if request.Mode != "letter" && request.Mode != "phrase" {
			http.Error(w, "mode must be 'letter' or 'phrase'", http.StatusBadRequest)
			return
		}

		var translate func(string) (string, error)
		if request.Mode == "letter" {
			translate = translateLetter
		} else {
			translate = translatePhrase
		}

		result, err := processText(request.Text, request.Mode, translate)
		if err != nil {
			log.Printf("process request: %v", err)
			http.Error(w, "could not process the supplied text", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, result); err != nil {
			log.Printf("write processing response: %v", err)
		}
	})

	log.Printf("Dictionary translator listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func processText(
	input string,
	mode string,
	translate func(string) (string, error),
) (string, error) {
	lines := strings.Split(input, "\n")
	var output strings.Builder

	for i, line := range lines {
		lineEnding := ""
		if strings.HasSuffix(line, "\r") {
			line = strings.TrimSuffix(line, "\r")
			lineEnding = "\r"
		}

		// Match the existing file-processing convention.
		if strings.HasPrefix(line, "xxxx") {
			output.WriteString(line)
		} else if mode == "letter" {
			translated, err := translate(line)
			if err != nil {
				return "", fmt.Errorf("translate line %d: %w", i+1, err)
			}
			output.WriteString(translated)
		} else {
			for _, token := range tokenPattern.FindAllString(line, -1) {
				if strings.TrimSpace(token) == "" {
					output.WriteString(token)
					continue
				}

				translated, err := translate(token)
				if err != nil {
					return "", fmt.Errorf("translate line %d: %w", i+1, err)
				}
				output.WriteString(translated)
			}
		}

		output.WriteString(lineEnding)
		if i < len(lines)-1 {
			output.WriteByte('\n')
		}
	}

	return output.String(), nil
}

func loadJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
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
