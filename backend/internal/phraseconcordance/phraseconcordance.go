
package phraseconcordance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

const UnknownTranslation = "\uFFFD"

type Concordance struct {
	Language Language               `json:"language"`
	Entries  map[string]PhraseEntry `json:"entries"`
}

type Language struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PhraseEntry struct {
	Lemma          string                    `json:"lemma"`
	Xlit           string                    `json:"xlit"`
	Pron           string                    `json:"pron"`
	PreferredPart  string                    `json:"preferred_speech_part"`
	Derivation     string                    `json:"derivation"`
	PreferredDef   string                    `json:"preferred_def"`
	PreferredTrans string                    `json:"preferred_trans"`
	Masoretic      map[string]MasoreticEntry `json:"masoretic_entries"`
}

type MasoreticEntry struct {
	Lemma      string  `json:"lemma"`
	Xlit       string  `json:"xlit"`
	Pron       string  `json:"pron"`
	SpeechPart string  `json:"speech_part"`
	Derivation string  `json:"derivation"`
	StrongsDef string  `json:"strongs_def"`
	KJVDef     string  `json:"kjv_def"`
	KJVCount   int     `json:"kjv_count"`
	KJVUsage   []Usage `json:"kjv_usage"`
}

type Usage struct {
	Translation string
	Count       int
}

func (u *Usage) UnmarshalJSON(data []byte) error {
	var values []json.RawMessage
	if err := json.Unmarshal(data, &values); err != nil {
		return fmt.Errorf("parse KJV usage: %w", err)
	}
	if len(values) < 2 {
		return fmt.Errorf("KJV usage must contain a translation and count")
	}
	if err := json.Unmarshal(values[0], &u.Translation); err != nil {
		return fmt.Errorf("parse KJV usage translation: %w", err)
	}
	if err := json.Unmarshal(values[1], &u.Count); err != nil {
		return fmt.Errorf("parse KJV usage count: %w", err)
	}
	return nil
}

// AffixFile supports individual prefixes or suffixes, top-level common
// combinations and the nested examples format.
type AffixFile struct {
	Language          json.RawMessage `json:"language"`
	Affixes           []Affix         `json:"-"`
	Prefixes          []Affix         `json:"prefixes"`
	Suffixes          []Affix         `json:"suffixes"`
	CommonCombinations []Combination  `json:"common_combinations"`
	Examples          AffixExamples   `json:"examples"`
}

// Affix represents an individual prefix or suffix.
type Affix struct {
	Name            string   `json:"name"`
	PartOfSpeech    string   `json:"part_of_speech"`
	SquareSymbol    string   `json:"square_symbol"`
	SquareUnicode   string   `json:"square_unicode"`
	PaleoSymbol     string   `json:"paleo_symbol"`
	PaleoUnicode    string   `json:"paleo_unicode"`
	Transliteration string   `json:"transliteration"`
	Meanings        []string `json:"meanings"`
	PreferredMeaning string  `json:"preferred_meaning"`
}

// Combination supports entries in common_combinations and examples.
type Combination struct {
	CombinationName       string `json:"combination_name"`
	Transliteration       string `json:"transliteration"`
	PreferredMeaning      string `json:"preferred_meaning"`
	ExampleWordModern     string `json:"example_word_modern"`
	ExampleWordPaleo      string `json:"example_word_paleo"`
	ExampleTransliteration string `json:"example_transliteration"`
	ExampleTranslation    string `json:"example_translation"`
}

type AffixExamples struct {
	CompoundPrefixes             []Combination `json:"compound_prefixes"`
	CompoundSuffixesAndFusions   []Combination `json:"compound_suffixes_and_fusions"`
}

// indexedAffix stores a matchable sequence and its translation.
type indexedAffix struct {
	Sequence        string
	PreferredMeaning string
	Name            string
	Kind            string
	Components      []string
}

func Load(path string) (Concordance, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Concordance{}, fmt.Errorf("read phrase concordance: %w", err)
	}

	var c Concordance
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&c); err != nil {
		return Concordance{}, fmt.Errorf("parse phrase concordance JSON: %w", err)
	}
	if len(c.Entries) == 0 {
		return Concordance{}, fmt.Errorf("phrase concordance contains no entries")
	}
	return c, nil
}

func Normalize(input string) string {
	var result strings.Builder

	for _, r := range strings.TrimSpace(input) {
		// Remove Hebrew vowel points and cantillation marks for matching.
		if unicode.In(r, unicode.Mn) && r >= '\u0591' && r <= '\u05C7' {
			continue
		}
		result.WriteRune(r)
	}

	return strings.Join(strings.Fields(result.String()), " ")
}

func BuildIndex(c Concordance) map[string]PhraseEntry {
	index := make(map[string]PhraseEntry, len(c.Entries))

	for _, entry := range c.Entries {
		key := Normalize(entry.Lemma)
		if key == "" {
			continue
		}
		if _, exists := index[key]; !exists {
			index[key] = entry
		}
	}

	return index
}

func Lookup(index map[string]PhraseEntry, input string) (string, bool) {
	entry, found := index[Normalize(input)]
	if !found {
		return UnknownTranslation, false
	}

	if preferred := strings.TrimSpace(entry.PreferredTrans); preferred != "" {
		return preferred, true
	}

	totals := make(map[string]int)
	for _, child := range entry.Masoretic {
		for _, usage := range child.KJVUsage {
			translation := strings.TrimSpace(usage.Translation)
			if translation == "" {
				continue
			}
			totals[translation] += usage.Count
		}
	}

	if len(totals) == 0 {
		return UnknownTranslation, true
	}

	translations := make([]string, 0, len(totals))
	for translation := range totals {
		translations = append(translations, translation)
	}
	sort.Strings(translations)

	bestCount := -1
	var winners []string

	for _, translation := range translations {
		count := totals[translation]
		if count > bestCount {
			bestCount = count
			winners = []string{translation}
		} else if count == bestCount {
			winners = append(winners, translation)
		}
	}

	return strings.Join(winners, "|"), true
}

// LoadAffixes reads individual affixes and supported combination formats.
// listName should be "prefixes" or "suffixes".
func LoadAffixes(path, listName string) (AffixFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return AffixFile{}, fmt.Errorf("read %s file: %w", listName, err)
	}

	var file AffixFile
	if err := json.Unmarshal(data, &file); err != nil {
		return AffixFile{}, fmt.Errorf("parse %s JSON: %w", listName, err)
	}

	switch listName {
	case "prefixes":
		file.Affixes = file.Prefixes
	case "suffixes":
		file.Affixes = file.Suffixes
	default:
		return AffixFile{}, fmt.Errorf("unsupported affix list %q", listName)
	}

	if len(file.Affixes) == 0 &&
		len(file.CommonCombinations) == 0 &&
		len(file.Examples.CompoundPrefixes) == 0 &&
		len(file.Examples.CompoundSuffixesAndFusions) == 0 {
		return AffixFile{}, fmt.Errorf("%s file contains no affixes or combinations", listName)
	}

	return file, nil
}

// BuildAffixIndex builds an index containing individual affixes and any
// combinations defined in the file. Compound character sequences are
// resolved against the individual affixes when a word is translated.
func BuildAffixIndex(file AffixFile) map[string]indexedAffix {
	index := make(map[string]indexedAffix)

	kind := "prefix"
	if len(file.Suffixes) > 0 || len(file.Affixes) > 0 && len(file.Prefixes) == 0 {
		kind = "suffix"
	}

	for _, affix := range file.Affixes {
		meaning := strings.TrimSpace(affix.PreferredMeaning)
		if meaning == "" {
			continue
		}

		for _, sequence := range []string{
			affix.SquareSymbol,
			affix.PaleoSymbol,
			affix.SquareUnicode,
			affix.PaleoUnicode,
		} {
			key := Normalize(sequence)
			if key == "" {
				continue
			}

			index[key] = indexedAffix{
				Sequence:         key,
				PreferredMeaning: meaning,
				Name:             affix.Name,
				Kind:             kind,
			}
		}
	}

	for _, combination := range file.CommonCombinations {
		addCombination(index, combination, kind)
	}

	for _, combination := range file.Examples.CompoundPrefixes {
		addCombination(index, combination, "prefix")
	}

	for _, combination := range file.Examples.CompoundSuffixesAndFusions {
		addCombination(index, combination, "suffix")
	}

	return index
}

func addCombination(index map[string]indexedAffix, combination Combination, kind string) {
	meaning := strings.TrimSpace(combination.PreferredMeaning)
	if meaning == "" {
		return
	}

	components := splitCombinationName(combination.CombinationName)
	item := indexedAffix{
		PreferredMeaning: meaning,
		Name:             combination.CombinationName,
		Kind:             kind,
		Components:       components,
	}

	// These example words are useful as exact standalone matches, but they
	// include the root word as well as the affix. They must not be treated
	// as the affix sequence itself.
	for _, example := range []string{
		combination.ExampleWordModern,
		combination.ExampleWordPaleo,
	} {
		key := Normalize(example)
		if key == "" {
			continue
		}

		exact := item
		exact.Sequence = key
		exact.Kind = "exact"
		index["exact:"+key] = exact
	}

	// The component names are resolved to actual characters when matching.
	index["combination:"+strings.ToLower(strings.TrimSpace(combination.CombinationName))] = item
}

func splitCombinationName(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil
	}

	parts := strings.Split(name, "+")
	result := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		// Remove explanatory text, e.g. "Vav (Conjunction)".
		if i := strings.Index(part, "("); i >= 0 {
			part = strings.TrimSpace(part[:i])
		}
		if part != "" {
			result = append(result, part)
		}
	}

	return result
}

// LookupWithAffixes tries an exact phrase first. Otherwise, it repeatedly
// consumes the longest matching prefix or suffix. It supports modern Hebrew
// and Paleo-Hebrew characters, plus combination examples from both files.
func LookupWithAffixes(
	phraseIndex map[string]PhraseEntry,
	prefixIndex map[string]indexedAffix,
	suffixIndex map[string]indexedAffix,
	input string,
) string {
	original := strings.TrimSpace(input)
	normalized := Normalize(original)
	if normalized == "" {
		return ""
	}

	if translation, found := Lookup(phraseIndex, normalized); found {
		return translation
	}

	// Standalone compound examples should translate as a whole before
	// attempting to strip individual affixes.
	if translation, found := lookupExactCombination(prefixIndex, normalized); found {
		return translation
	}
	if translation, found := lookupExactCombination(suffixIndex, normalized); found {
		return translation
	}

	// Build one component-name lookup from both files so combinations can
	// resolve letters defined in either file.
	componentIndex := make(map[string]indexedAffix)
	for _, index := range []map[string]indexedAffix{prefixIndex, suffixIndex} {
		for key, entry := range index {
			if strings.HasPrefix(key, "combination:") || strings.HasPrefix(key, "exact:") {
				continue
			}
			if entry.Name != "" {
				componentIndex[strings.ToLower(strings.TrimSpace(entry.Name))] = entry
			}
		}
	}

	remaining := normalized
	var translations []string

	for remaining != "" {
		if translation, found := Lookup(phraseIndex, remaining); found {
			if translation != "" && translation != UnknownTranslation {
				translations = append(translations, translation)
			} else {
				translations = append(translations, remaining)
			}
			remaining = ""
			break
		}

		if sequence, meaning, ok := longestCombinationMatch(prefixIndex, componentIndex, remaining, true); ok {
			translations = append(translations, meaning)
			remaining = strings.TrimPrefix(remaining, sequence)
			continue
		}

		if sequence, meaning, ok := longestAffixMatch(prefixIndex, remaining, true); ok {
			translations = append(translations, meaning)
			remaining = strings.TrimPrefix(remaining, sequence)
			continue
		}

		if sequence, meaning, ok := longestCombinationMatch(suffixIndex, componentIndex, remaining, false); ok {
			translations = append(translations, meaning)
			remaining = strings.TrimSuffix(remaining, sequence)
			continue
		}

		if sequence, meaning, ok := longestAffixMatch(suffixIndex, remaining, false); ok {
			translations = append(translations, meaning)
			remaining = strings.TrimSuffix(remaining, sequence)
			continue
		}

		// No match remains. Preserve the unmatched part instead of replacing
		// it with an unknown-character placeholder.
		translations = append(translations, remaining)
		break
	}

	if len(translations) == 0 {
		return original
	}
	return strings.Join(translations, " ")
}

func lookupExactCombination(index map[string]indexedAffix, input string) (string, bool) {
	key := Normalize(input)
	entry, found := index["exact:"+key]
	if !found || entry.PreferredMeaning == "" {
		return "", false
	}
	return entry.PreferredMeaning, true
}

func longestAffixMatch(
	index map[string]indexedAffix,
	remaining string,
	prefix bool,
) (string, string, bool) {
	var bestSequence string
	var bestMeaning string

	for key, entry := range index {
		if strings.HasPrefix(key, "exact:") || strings.HasPrefix(key, "combination:") {
			continue
		}

		sequence := Normalize(entry.Sequence)
		if sequence == "" {
			sequence = key
		}
		if sequence == "" || sequence == remaining {
			// Whole-word matches are handled separately to avoid stripping
			// the entire word as an ordinary affix.
			continue
		}

		matched := strings.HasPrefix(remaining, sequence)
		if !prefix {
			matched = strings.HasSuffix(remaining, sequence)
		}
		if matched && utf8.RuneCountInString(sequence) > utf8.RuneCountInString(bestSequence) {
			bestSequence = sequence
			bestMeaning = entry.PreferredMeaning
		}
	}

	return bestSequence, bestMeaning, bestSequence != ""
}

func longestCombinationMatch(
	index map[string]indexedAffix,
	componentIndex map[string]indexedAffix,
	remaining string,
	prefix bool,
) (string, string, bool) {
	var bestSequence string
	var bestMeaning string

	for key, combination := range index {
		if !strings.HasPrefix(key, "combination:") || len(combination.Components) == 0 {
			continue
		}

		sequence := resolveCombinationSequence(combination.Components, componentIndex)
		if sequence == "" || sequence == remaining {
			continue
		}

		matched := strings.HasPrefix(remaining, sequence)
		if !prefix {
			matched = strings.HasSuffix(remaining, sequence)
		}

		if matched && utf8.RuneCountInString(sequence) > utf8.RuneCountInString(bestSequence) {
			bestSequence = sequence
			bestMeaning = combination.PreferredMeaning
		}
	}

	return bestSequence, bestMeaning, bestSequence != ""
}

func resolveCombinationSequence(
	components []string,
	componentIndex map[string]indexedAffix,
) string {
	var modern strings.Builder
	var paleo strings.Builder
	var usePaleo bool

	for _, component := range components {
		entry, found := componentIndex[strings.ToLower(strings.TrimSpace(component))]
		if !found {
			return ""
		}

		sequence := entry.Sequence
		if sequence == "" {
			return ""
		}

		if strings.ContainsRune(sequence, '\U00010900') ||
			strings.ContainsRune(sequence, '\U00010920') {
			usePaleo = true
		}

		modern.WriteString(entry.Sequence)
		paleo.WriteString(entry.Sequence)
	}

	if usePaleo {
		return Normalize(paleo.String())
	}
	return Normalize(modern.String())
}
