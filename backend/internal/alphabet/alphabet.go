
package alphabet

import (
	"fmt"
	"strconv"
	"strings"
	"sort"
	"unicode"
	"unicode/utf8"
)

const UnknownCharacter = "\uFFFD"

type Chart struct {
	Document Document `json:"document"`
}

type Document struct {
	Name  string       `json:"name"`
	Types map[string]ScriptType `json:"types"`
	Entries []Entry    `json:"entries"`
}

type ScriptType struct {
	Name        string `json:"name"`
	Period      string `json:"period"`
	Description string `json:"description"`
}

type Entry struct {
	LetterName   string   `json:"letter_name"`
	Meaning      string   `json:"meaning"`
	Pictograph   string   `json:"pictograph"`
	PaleoUnicode string   `json:"paleo_unicode"`
	PaleoSymbol  string   `json:"paleo_symbol"`
	SquareUnicode []string `json:"square_unicode"`
	SquareSymbol  []string `json:"square_symbol"`
}

// Normalize converts Unicode code points such as U+05D0 into
// the same representation used for literal characters.
func Normalize(input string) string {
	input = strings.TrimSpace(input)

	if strings.HasPrefix(strings.ToUpper(input), "U+") {
		hex := input[2:]
		value, err := strconv.ParseInt(hex, 16, 32)
		if err == nil && value >= 0 {
			return strings.ToLower(string(rune(value)))
		}
	}

	return strings.ToLower(input)
}

// BuildIndex maps letter names, literal symbols and Unicode code
// points to the canonical letter name.
func BuildIndex(chart Chart) map[string]string {
	index := make(map[string]string)

	for _, entry := range chart.Document.Entries {
		name := strings.ToLower(strings.TrimSpace(entry.LetterName))
		if name == "" {
			continue
		}

		index[Normalize(name)] = entry.LetterName

		if entry.PaleoSymbol != "" {
			index[Normalize(entry.PaleoSymbol)] = entry.LetterName
		}

		if entry.PaleoUnicode != "" {
			index[Normalize(entry.PaleoUnicode)] = entry.LetterName
		}

		for _, symbol := range entry.SquareSymbol {
			index[Normalize(symbol)] = entry.LetterName
		}

		for _, codePoint := range entry.SquareUnicode {
			index[Normalize(codePoint)] = entry.LetterName
		}
	}

	return index
}

// Lookup resolves a name, symbol or Unicode code point.
func Lookup(index map[string]string, input string) (string, bool) {
	key := Normalize(input)
	name, found := index[key]

	if !found {
		return "", false
	}

	return name, true
}

// Validate checks whether the chart contains any entries.
func (chart Chart) Validate() error {
	if len(chart.Document.Entries) == 0 {
		return fmt.Errorf("alphabet chart contains no entries")
	}

	return nil
}

// ResolveSequence splits an input into letter names.
// It supports concatenated names such as AlefBet, names separated
// by spaces, and sequences of Unicode characters such as אב or 𐤀𐤁.
func ResolveSequence(index map[string]string, input string) ([]string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, fmt.Errorf("input is empty")
	}

	// Build a list of recognised names, longest first.
	var names []string
	for key, name := range index {
		// Only use alphabetic names for concatenated-name matching.
		if key != "" && unicode.IsLetter([]rune(key)[0]) &&
			isASCIIName(key) && !containsUnicodeSymbol(key) {
			names = append(names, key)
		}
		_ = name
	}

	sort.Slice(names, func(i, j int) bool {
		return len(names[i]) > len(names[j])
	})

	remaining := strings.ToLower(input)
	var result []string

	for len(remaining) > 0 {
		// Ignore spaces between names or characters.
		if strings.HasPrefix(remaining, " ") {
			remaining = strings.TrimLeft(remaining, " ")
			continue
		}

		matched := false

		// Try matching a full letter name, longest first.
		for _, name := range names {
			if strings.HasPrefix(remaining, name) {
				result = append(result, index[name])
				remaining = remaining[len(name):]
				matched = true
				break
			}
		}

		if matched {
			continue
		}

		// If no name matches, try one Unicode character.
		r, size := utf8.DecodeRuneInString(remaining)
		if r == utf8.RuneError && size == 0 {
			break
		}

		character := string(r)
		letterName, found := Lookup(index, character)

		if !found {
			result = append(result, UnknownCharacter)
			remaining = remaining[size:]
			continue
		}

		result = append(result, letterName)
		remaining = remaining[size:]
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no letters found")
	}

	return result, nil
}

func isASCIIName(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII || !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func containsUnicodeSymbol(s string) bool {
	for _, r := range s {
		if r > unicode.MaxASCII {
			return true
		}
	}
	return false
}
