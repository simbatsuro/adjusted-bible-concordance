package phraseconcordance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"log"
	"path/filepath"
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
	HKID           string                    `json:"-"`
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
	Language           json.RawMessage `json:"language"`
	Affixes            []Affix         `json:"-"`
	Prefixes           []Affix         `json:"prefixes"`
	Suffixes           []Affix         `json:"suffixes"`
	CommonCombinations []Combination   `json:"common_combinations"`
	Examples           AffixExamples   `json:"examples"`
}

// Affix represents an individual prefix or suffix.
type Affix struct {
	Name             string   `json:"name"`
	PartOfSpeech     string   `json:"part_of_speech"`
	SquareSymbol     string   `json:"square_symbol"`
	SquareUnicode    string   `json:"square_unicode"`
	PaleoSymbol      string   `json:"paleo_symbol"`
	PaleoUnicode     string   `json:"paleo_unicode"`
	Transliteration  string   `json:"transliteration"`
	Meanings         []string `json:"meanings"`
	PreferredMeaning string   `json:"preferred_meaning"`
}

// Combination supports entries in common_combinations and examples.
type Combination struct {
	CombinationName        string `json:"combination_name"`
	Transliteration        string `json:"transliteration"`
	PreferredMeaning       string `json:"preferred_meaning"`
	ExampleWordModern      string `json:"example_word_modern"`
	ExampleWordPaleo       string `json:"example_word_paleo"`
	ExampleTransliteration string `json:"example_transliteration"`
	ExampleTranslation     string `json:"example_translation"`
}

type AffixExamples struct {
	CompoundPrefixes           []Combination `json:"compound_prefixes"`
	CompoundSuffixesAndFusions []Combination `json:"compound_suffixes_and_fusions"`
}

// indexedAffix stores a matchable sequence and its translation.
type indexedAffix struct {
	Sequence         string
	PreferredMeaning string
	Name             string
	Kind             string
	Components       []string
}

// TranslationPreferenceFile stores user-selected interpretations separately
// from the lexical concordance and affix data.
type TranslationPreferenceFile struct {
	Preferences []TranslationPreference `json:"preferences"`
}

type TranslationPreference struct {
	Hebrew               string   `json:"hebrew"`
	PreferredTranslation string   `json:"preferred_translation"`
	PreferredMatch       string   `json:"preferred_match"`
	Alternatives         []string `json:"alternatives"`
	Enabled              bool     `json:"enabled"`
	Notes                string   `json:"notes"`
}

func LoadPrefs(path string) (TranslationPreferenceFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return TranslationPreferenceFile{}, fmt.Errorf("read phrase concordance: %w", err)
	}

	var translationFile TranslationPreferenceFile
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&translationFile); err != nil {
		return TranslationPreferenceFile{}, fmt.Errorf("parse phrase concordance JSON: %w", err)
	}
	if len(translationFile.Preferences) == 0 {
		return TranslationPreferenceFile{}, fmt.Errorf("phrase concordance contains no entries")
	}
	return translationFile, nil
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
		// Final forms are equivalent to their medial consonants for lookup.
		switch r {
		case 'ך':
			r = 'כ'
		case 'ם':
			r = 'מ'
		case 'ן':
			r = 'נ'
		case 'ף':
			r = 'פ'
		case 'ץ':
			r = 'צ'
		}
		result.WriteRune(r)
	}
	return strings.Join(strings.Fields(result.String()), " ")
}

func BuildIndex(c Concordance) map[string]PhraseEntry {
	index := make(map[string]PhraseEntry, len(c.Entries))
	for hkID, entry := range c.Entries {
		key := Normalize(entry.Lemma)
		if key == "" {
			continue
		}
		entry.HKID = hkID
		// Prefer a direct consonantal lemma match; do not let map iteration
		// order determine which HK group wins when duplicate lemmas exist.
		if previous, exists := index[key]; exists {
			if hkID < previous.HKID {
				continue
			}
		}
		index[key] = entry
	}
	return index
}

func Lookup(index map[string]PhraseEntry, input string) (string, bool) {
	entry, found := index[Normalize(input)]
	if !found {
		return UnknownTranslation, false
	}

	// Only preferred_trans is used as the translation.
	translation := strings.TrimSpace(entry.PreferredTrans)
	if translation == "" {
		return UnknownTranslation, false
	}

	// Join the English words belonging to this Hebrew word.
	translation = translationBlock(translation)

	//START FOR_DEBUGGING_BLOCK
	// Keep the adjusted HK reference, not the individual H references.
	// if entry.HKID != "" {
	// 	translation = entry.HKID + ":" + translation
	// }

	// Show the adjusted HK identity and the H entries grouped beneath it.
	// if entry.HKID != "" {
	// 	ids := make([]string, 0, len(entry.Masoretic))
	// 	for id := range entry.Masoretic {
	// 		ids = append(ids, id)
	// 	}
	// 	sort.Strings(ids)
	// 	if len(ids) > 0 {
	// 		translation = entry.HKID + " [" + strings.Join(ids, ",") + "] " + translation
	// 	} else {
	// 		translation = entry.HKID + " " + translation
	// 	}
	// }
	//END FOR_DEBUGGING_BLOCK

	return translation, true
}

// translationBlock joins all English words belonging to one Hebrew word.
// Spaces within a translation become underscores.
func translationBlock(input string) string {
	words := strings.Fields(strings.TrimSpace(input))
	return strings.Join(words, "_")
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

// LookupWithAffixes keeps the original lookup interface for callers that do
// not use translation preferences.
func LookupWithAffixes(
	phraseIndex map[string]PhraseEntry,
	prefixIndex map[string]indexedAffix,
	suffixIndex map[string]indexedAffix,
	input string,
) string {
	direct, alternatives := findAffixTranslations(
		phraseIndex, prefixIndex, suffixIndex, input,
	)
	return formatLookupResult(direct, alternatives)
}

// LookupWithAffixesAndPreferences applies user-managed preferences when a
// word has multiple valid translations and records those alternatives.
func LookupWithAffixesAndPreferences(
	phraseIndex map[string]PhraseEntry,
	prefixIndex map[string]indexedAffix,
	suffixIndex map[string]indexedAffix,
	input string,
	translationFile TranslationPreferenceFile,
) (string, error) {
	// if translationFile == "" {
	// 	return "", fmt.Errorf("translation preferences path is empty")
	// }

	direct, alternatives := findAffixTranslations(
		phraseIndex, prefixIndex, suffixIndex, input,
	)

	if len(alternatives) < 2 {
		return formatLookupResult(direct, alternatives), nil
	}

	word := Normalize(input)
	preferred, err := applyTranslationPreference(word, alternatives, translationFile)
	if err != nil {
		return "", err
	}
	return formatLookupResult("", preferred), nil
}

// findAffixTranslations only finds translations. It does not read or write
// preference files, keeping matching separate from preference management.
// A direct result is returned for exact matches or when no match is found.
func findAffixTranslations(
	phraseIndex map[string]PhraseEntry,
	prefixIndex map[string]indexedAffix,
	suffixIndex map[string]indexedAffix,
	input string,
) (string, []string) {
	original := strings.TrimSpace(input)
	word := Normalize(original)

	if word == "" {
		return "", nil
	}

	// Exact lexical and affix-combination matches take priority.
	if translation, ok := Lookup(phraseIndex, word); ok {
		return translation, nil
	}
	if translation, ok := lookupExactCombination(prefixIndex, word); ok {
		return translationBlock(translation), nil
	}
	if translation, ok := lookupExactCombination(suffixIndex, word); ok {
		return translationBlock(translation), nil
	}

	type result struct {
		score int
		text  string
	}

	var results []result
	seen := make(map[string]bool)

	// Explore all valid prefix and suffix combinations. Each remaining stem
	// is checked against the HK concordance.
	var explore func(string, []string, []string, int, int)
	explore = func(
		remaining string,
		prefixMeanings []string,
		suffixMeanings []string,
		prefixCount int,
		suffixCount int,
	) {
		if remaining == "" {
			return
		}

		if lexical, found := Lookup(phraseIndex, remaining); found && lexical != UnknownTranslation {
			translation := formatAffixedTranslation(
				lexical,
				strings.Join(prefixMeanings, " "),
				strings.Join(suffixMeanings, " "),
			)
			if !seen[translation] {
				seen[translation] = true
				results = append(results, result{
					score: prefixCount + suffixCount,
					text:  translation,
				})
			}
		}

		for _, prefix := range affixCandidates(prefixIndex, remaining, true) {
			next := strings.TrimPrefix(remaining, prefix.sequence)
			if next == "" || next == remaining {
				continue
			}
			nextMeanings := append(append([]string{}, prefixMeanings...), prefix.meaning)
			explore(next, nextMeanings, suffixMeanings, prefixCount+1, suffixCount)
		}

		for _, suffix := range affixCandidates(suffixIndex, remaining, false) {
			next := strings.TrimSuffix(remaining, suffix.sequence)
			if next == "" || next == remaining {
				continue
			}
			nextMeanings := append(append([]string{}, suffixMeanings...), suffix.meaning)
			explore(next, prefixMeanings, nextMeanings, prefixCount, suffixCount+1)
		}
	}

	explore(word, nil, nil, 0, 0)
	if len(results) == 0 {
		return original, nil
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].score != results[j].score {
			return results[i].score > results[j].score
		}
		return results[i].text < results[j].text
	})

	translations := make([]string, 0, len(results))
	for _, item := range results {
		translations = append(translations, item.text)
	}
	if len(translations) == 1 {
		return translations[0], nil
	}
	return "", translations
}

// formatLookupResult formats either a direct translation or a list of
// alternative interpretations.
func formatLookupResult(direct string, alternatives []string) string {
	if len(alternatives) == 0 {
		return direct
	}
	if len(alternatives) == 1 {
		return alternatives[0]
	}
	return "[" + strings.Join(alternatives, " | ") + "]"
}

// applyTranslationPreference refreshes the alternatives for this Hebrew word.
// New records remain disabled until the user selects and enables a preference.
func applyTranslationPreference(
	hebrew string,
	alternatives []string,
	translationFile TranslationPreferenceFile,	
) ([]string, error) {
	// index := make(map[string]TranslationPreference, len(translationFile.Preferences))
	
	for hebID, preference := range translationFile.Preferences {
		if preference.Hebrew == hebrew {
			key := Normalize(preference.Hebrew)
			if key == "" {
				continue
			}

			//Do something
			log.Printf("%s",hebID)

			if preference.Enabled && strings.TrimSpace(preference.PreferredTranslation) != "" {
				ordered := append([]string{}, preference.PreferredTranslation)
				return ordered, nil
			}
		}
	}

	ordered := append([]string{}, alternatives...)

	// if err := saveTranslationPreferences(path, file); err != nil {
	// 	return nil, err
	// }
	
	return ordered, nil
}

func loadTranslationPreferences(path string) (TranslationPreferenceFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return TranslationPreferenceFile{
				Preferences: []TranslationPreference{},
			}, nil
		}
		return TranslationPreferenceFile{}, fmt.Errorf("read translation preferences: %w", err)
	}

	var file TranslationPreferenceFile
	if err := json.Unmarshal(data, &file); err != nil {
		return TranslationPreferenceFile{}, fmt.Errorf("parse translation preferences JSON: %w", err)
	}
	if file.Preferences == nil {
		file.Preferences = []TranslationPreference{}
	}
	return file, nil
}

func saveTranslationPreferences(path string, file TranslationPreferenceFile) error {
	for i := range file.Preferences {
		if file.Preferences[i].Alternatives == nil {
			file.Preferences[i].Alternatives = []string{}
		}
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("encode translation preferences: %w", err)
	}

	directory := filepath.Dir(path)
	tempFile, err := os.CreateTemp(directory, ".translation_preferences-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary translation preferences file: %w", err)
	}
	tempPath := tempFile.Name()

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return fmt.Errorf("write translation preferences: %w", err)
	}
	if err := tempFile.Sync(); err != nil {
		tempFile.Close()
		os.Remove(tempPath)
		return fmt.Errorf("sync translation preferences: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("close translation preferences: %w", err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("replace translation preferences: %w", err)
	}
	return nil
}

// formatAffixedTranslation keeps each Hebrew word's translation together.
// Prefixes come first, ordinary suffixes precede the lexical translation,
// and plural -es is appended to the lexical translation.
func formatAffixedTranslation(
	lexicalTranslation, prefixTranslation, suffixTranslation string,
) string {
	lexicalTranslation = translationBlock(lexicalTranslation)
	prefixTranslation = translationBlock(prefixTranslation)
	suffixTranslation = translationBlock(suffixTranslation)

	parts := make([]string, 0, 3)

	// Prefix translation comes first.
	if prefixTranslation != "" {
		parts = append(parts, prefixTranslation)
	}

	// Plural ending is appended to the lexical translation.
	if suffixTranslation == "-es" {
		lexicalTranslation += "(-es)"
	} else if suffixTranslation != "" {
		// Ordinary suffix translation comes before the lexical word.
		parts = append(parts, suffixTranslation)
	}

	if lexicalTranslation != "" {
		parts = append(parts, lexicalTranslation)
	}

	return strings.Join(parts, "_")
}

type affixCandidate struct{ sequence, meaning string }

func affixCandidates(index map[string]indexedAffix, word string, prefix bool) []affixCandidate {
	var found []affixCandidate
	seen := map[string]bool{}
	for key, entry := range index {
		if strings.HasPrefix(key, "exact:") || strings.HasPrefix(key, "combination:") {
			continue
		}
		seq := Normalize(entry.Sequence)
		if seq == "" {
			seq = Normalize(key)
		}
		if seq == "" || seq == word {
			continue
		}
		matched := strings.HasPrefix(word, seq)
		if !prefix {
			matched = strings.HasSuffix(word, seq)
		}
		if !matched {
			continue
		}
		id := seq + "\x00" + entry.PreferredMeaning
		if !seen[id] {
			seen[id] = true
			found = append(found, affixCandidate{seq, entry.PreferredMeaning})
		}
	}
	// Longer affixes first, but retain all valid alternatives.
	sort.Slice(found, func(i, j int) bool {
		return utf8.RuneCountInString(found[i].sequence) > utf8.RuneCountInString(found[j].sequence)
	})
	return found
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
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