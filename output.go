package main

import (
	"bytes"
	"fmt"
	"os"
)

// generateToFile renders into memory and writes the output file only after a
// successful run, so a failed run leaves an existing file as it was instead
// of truncating it.
func generateToFile(gen *Generator, dirs []string, outFile string) error {
	var buf bytes.Buffer
	if err := gen.GenerateDirs(dirs, &buf); err != nil {
		return fmt.Errorf("failed to generate: %w", err)
	}
	if err := os.WriteFile(outFile, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	return nil
}
