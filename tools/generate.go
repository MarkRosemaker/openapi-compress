package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/MarkRosemaker/fsutil/osutil"
	"github.com/MarkRosemaker/openapi"
	compress "github.com/MarkRosemaker/openapi-compress"
)

func main() {
	if err := run(context.Background()); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	if err := copyPreviousStep(); err != nil {
		return err
	}

	entries, err := os.ReadDir("testdata")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		doc, err := openapi.LoadFromFile(filepath.Join("testdata", entry.Name(), "openapi.json"))
		if err != nil {
			return err
		}

		if err := compress.Document(doc, compress.Config{MinSimilarity: 0.8}); err != nil {
			return err
		}

		doc.Components.SortMaps()

		if err := doc.WriteToFile(filepath.Join("testdata", entry.Name(), "golden.json")); err != nil {
			return err
		}
	}

	return nil
}

func copyPreviousStep() error {
	const (
		flattenDir = "../openapi-flatten/testdata"
		enrichDir  = "../openapi-enrich/testdata"
	)

	for srcDir, names := range map[string][2]string{
		flattenDir: {"golden.json", "openapi.json"},
		enrichDir:  {"interactions.json", "interactions.json"},
	} {
		entries, err := os.ReadDir(srcDir)
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				return nil
			}

			return fmt.Errorf("reading folder: %w", err)
		}

		for _, e := range entries {
			if e.Name() == ".DS_Store" {
				continue
			}

			src := filepath.Join(srcDir, e.Name(), names[0])
			dst := filepath.Join("testdata", e.Name(), names[1])

			// a recording without interactions has nothing for the test to check against
			if empty, err := isEmptyArray(src); err != nil {
				return err
			} else if empty {
				if err := os.Remove(dst); err != nil && !errors.Is(err, fs.ErrNotExist) {
					return err
				}

				continue
			}

			if err := osutil.Copy(src, dst); err != nil {
				return fmt.Errorf("copying file: %w", err)
			}
		}
	}

	return nil
}

// isEmptyArray reports whether the JSON file at path holds an empty array.
func isEmptyArray(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}

		return false, err
	}

	return string(bytes.TrimSpace(data)) == "[]", nil
}
