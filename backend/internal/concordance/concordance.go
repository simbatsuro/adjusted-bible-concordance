
package concordance

import "strings"

type Concordance struct {
	Language struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	} `json:"language"`

	Entries map[string]LetterEntry `json:"entries"`
}

type LetterEntry struct {
	LetterName string   `json:"letter_name"`
	Meaning    []string `json:"meaning"`
	PreferredMeaning string   `json:"preferred_meaning"`
	Gematria   []int    `json:"gematria"`
}

// BuildLetterIndex creates an index using each letter's name.
func BuildLetterIndex(data Concordance) map[string]LetterEntry {
	index := make(map[string]LetterEntry, len(data.Entries))

	for _, entry := range data.Entries {
		key := strings.ToLower(strings.TrimSpace(entry.LetterName))

		if key != "" {
			index[key] = entry
		}
	}

	return index
}

// LookupLetter finds a letter by name, ignoring case and surrounding spaces.
func LookupLetter(
	index map[string]LetterEntry,
	name string,
) (LetterEntry, bool) {
	key := strings.ToLower(strings.TrimSpace(name))
	entry, found := index[key]

	return entry, found
}
