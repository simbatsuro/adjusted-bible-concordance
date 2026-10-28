
package fileprocessor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const SkipPrefix = "xxxx"

func ProcessFile(
	inputPath string,
	outputDir string,
	translate func(string) (string, error),
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

	outputPath := filepath.Join(
		outputDir,
		name+".output"+extension,
	)

	output, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("create output file: %w", err)
	}

	writer := bufio.NewWriter(output)
	scanner := bufio.NewScanner(input)

	// Allow lines longer than Scanner's default limit.
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		result := line

		// Copy lines starting with "xxxx" without translating.
		if !strings.HasPrefix(line, SkipPrefix) {
			result, err = translate(line)
			if err != nil {
				output.Close()
				return "", fmt.Errorf("translate line: %w", err)
			}
		}

		if _, err := writer.WriteString(result + "\n"); err != nil {
			output.Close()
			return "", fmt.Errorf("write output: %w", err)
		}
	}

	if err := scanner.Err(); err != nil {
		output.Close()
		return "", fmt.Errorf("read input: %w", err)
	}

	if err := writer.Flush(); err != nil {
		output.Close()
		return "", fmt.Errorf("flush output: %w", err)
	}

	if err := output.Close(); err != nil {
		return "", fmt.Errorf("close output: %w", err)
	}

	return outputPath, nil
}