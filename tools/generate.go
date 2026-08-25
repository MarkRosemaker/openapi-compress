package main

import (
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
	const flattenDir = "../openapi-flatten/testdata"
	const enrichDir = "../openapi-enrich/testdata"

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
			if err := osutil.Copy(
				filepath.Join(srcDir, e.Name(), names[0]),
				filepath.Join("testdata", e.Name(), names[1]),
			); err != nil {
				return fmt.Errorf("copying file: %w", err)
			}
		}

	}

	return nil
}
