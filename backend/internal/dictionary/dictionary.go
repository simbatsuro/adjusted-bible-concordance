
package dictionary

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Entry struct {
	Word         string   `json:"word"`
	PartOfSpeech string   `json:"partOfSpeech"`
	Definitions  []string `json:"definitions"`
}

type Dictionary struct {
	entries map[string]Entry
}

func Load(path string) (*Dictionary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read dictionary: %w", err)
	}

	var items []Entry
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("parse dictionary JSON: %w", err)
	}

	d := &Dictionary{
		entries: make(map[string]Entry, len(items)),
	}

	for _, item := range items {
		key := strings.ToLower(strings.TrimSpace(item.Word))
		if key == "" {
			return nil, fmt.Errorf("dictionary contains an empty word")
		}
		if _, exists := d.entries[key]; exists {
			return nil, fmt.Errorf("duplicate dictionary word: %s", key)
		}

		d.entries[key] = item
	}

	return d, nil
}

func (d *Dictionary) Lookup(word string) (Entry, bool) {
	key := strings.ToLower(strings.TrimSpace(word))
	entry, ok := d.entries[key]
	return entry, ok
}
