
package fileprocessor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const SkipPrefix = "xxxx"

var wordOrWhitespace = regexp.MustCompile(`\S+|\s+`)

// ProcessLetterFile translates each non-skipped line using the letter
// concordance and writes the result to a separate letter output file.
func ProcessLetterFile(
	inputPath string,
	outputDir string,
	translate func(string) (string, error),
) (string, error) {
	return processFile(inputPath, outputDir, ".letter", translate, false)
}

// ProcessPhraseFile translates each non-whitespace token separately using
// the phrase concordance. It preserves spaces and tabs between tokens.
func ProcessPhraseFile(
	inputPath string,
	outputDir string,
	translate func(string) (string, error),
) (string, error) {
	return processFile(inputPath, outputDir, ".phrase", translate, true)
}

func processFile(
	inputPath string,
	outputDir string,
	suffix string,
	translate func(string) (string, error),
	tokenMode bool,
) (string, error) {
	input, err := os.Open(inputPath)
	if err != nil {
		return "", fmt.Errorf("open input file: %w", err)
	}
	defer input.Close()

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}

	filename := filepath.Base(inputPath)
	extension := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, extension)

	// Keep the original extension while separating letter and phrase outputs.
	outputPath := filepath.Join(outputDir, name+suffix+extension)

	output, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("create output file: %w", err)
	}

	closed := false
	defer func() {
		if !closed {
			_ = output.Close()
		}
	}()

	writer := bufio.NewWriter(output)
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		result := line

		// Lines beginning with "xxxx" are copied without translation.
		if !strings.HasPrefix(line, SkipPrefix) {
			if tokenMode {
				result, err = translateTokens(line, translate)
			} else {
				result, err = translate(line)
			}

			if err != nil {
				return "", fmt.Errorf("process input line: %w", err)
			}
		}

		if _, err := writer.WriteString(result + "\n"); err != nil {
			return "", fmt.Errorf("write output: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}

	if err := writer.Flush(); err != nil {
		return "", fmt.Errorf("flush output: %w", err)
	}

	if err := output.Close(); err != nil {
		closed = true
		return "", fmt.Errorf("close output: %w", err)
	}

	closed = true
	return outputPath, nil
}

func translateTokens(
	line string,
	translate func(string) (string, error),
) (string, error) {
	parts := wordOrWhitespace.FindAllString(line, -1)

	var result strings.Builder

	for _, part := range parts {
		// Keep whitespace exactly as it appears between tokens.
		if strings.TrimSpace(part) == "" {
			result.WriteString(part)
			continue
		}

		translated, err := translate(part)
		if err != nil {
			return "", err
		}

		result.WriteString(translated)
	}

	return result.String(), nil
}
