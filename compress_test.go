package compress_test

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/MarkRosemaker/openapi"
	compress "github.com/MarkRosemaker/openapi-compress"
	enrich "github.com/MarkRosemaker/openapi-enrich"
	"github.com/MarkRosemaker/openapi-enrich/cassette"
)

// TestDocument_Golden compresses testdata/openapi.json three times over and must get testdata/golden.json each time.
// The document is what openapi-enrich and openapi-flatten make of testdata/interactions.json, with components added
// by hand where no recording would make them, so enriching the result with the recordings again must change nothing:
// compressing lost none of what they show. All three files are edited by hand.
func TestDocument_Golden(t *testing.T) {
	t.Parallel()

	doc, err := openapi.LoadFromFile("testdata/openapi.json")
	if err != nil {
		t.Fatal(err)
	}

	want, err := os.ReadFile("testdata/golden.json")
	if err != nil {
		t.Fatal(err)
	}

	for run := range 3 {
		if err := compress.Document(doc, compress.Config{MinSimilarity: 0.8}); err != nil {
			t.Fatalf("run %d: %v", run+1, err)
		}

		if err := doc.Validate(); err != nil {
			t.Fatalf("run %d: %v", run+1, err)
		}

		if line, ok := firstDifference(toJSON(t, doc), want); !ok {
			t.Fatalf("run %d: golden.json %s", run+1, line)
		}
	}

	interactions, err := cassette.InteractionsReadFile("testdata/interactions.json")
	if err != nil {
		t.Fatal(err)
	}

	if err := enrich.Enrich(doc, interactions); err != nil {
		t.Fatal(err)
	}

	if line, ok := firstDifference(toJSON(t, doc), want); !ok {
		t.Fatalf("enriched again, golden.json %s", line)
	}
}

// toJSON is doc as it is written, with its components sorted.
func toJSON(t *testing.T, doc *openapi.Document) []byte {
	t.Helper()

	doc.Components.SortMaps()

	b, err := doc.ToJSON()
	if err != nil {
		t.Fatal(err)
	}

	got := append(b, '\n')

	return got
}

// firstDifference describes the first line in which got and want differ, if any.
func firstDifference(got, want []byte) (string, bool) {
	if bytes.Equal(got, want) {
		return "", true
	}

	gotLines, wantLines := bytes.Split(got, []byte("\n")), bytes.Split(want, []byte("\n"))
	for i := range min(len(gotLines), len(wantLines)) {
		if !bytes.Equal(gotLines[i], wantLines[i]) {
			return fmt.Sprintf("line %d: got %s, want %s", i+1, bytes.TrimSpace(gotLines[i]), bytes.TrimSpace(wantLines[i])), false
		}
	}

	return fmt.Sprintf("has %d lines, got %d", len(wantLines), len(gotLines)), false
}
