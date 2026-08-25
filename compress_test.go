package compress_test

import (
	"bytes"
	"embed"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/MarkRosemaker/jsonutil"
	"github.com/MarkRosemaker/openapi"
	compress "github.com/MarkRosemaker/openapi-compress"
	enrich "github.com/MarkRosemaker/openapi-enrich"
	"github.com/MarkRosemaker/openapi-enrich/cassette"
)

var (
	//go:embed testdata
	testdata embed.FS

	jsonOpts = json.JoinOptions(
		jsontext.Multiline(true),
		json.WithMarshalers(json.MarshalToFunc(jsonutil.HTTPHeaderMarshal)),
		json.WithUnmarshalers(json.UnmarshalFromFunc(jsonutil.HTTPHeaderUnmarshal)),
	)
)

func TestCompress_TestData(t *testing.T) {
	entries, err := testdata.ReadDir("testdata")
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range entries {
		t.Run(tc.Name(), func(t *testing.T) {
			f, err := testdata.Open(filepath.Join("testdata", tc.Name(), "openapi.json"))
			if err != nil {
				t.Fatal(err)
			}
			defer f.Close() //nolint

			doc, err := openapi.LoadFromReader(f)
			if err != nil {
				t.Fatal(err)
			}

			wantDoc, err := testdata.ReadFile(filepath.Join("testdata", tc.Name(), "golden.json"))
			if err != nil {
				t.Fatal(err)
			}

			for it := range 3 {
				t.Run(fmt.Sprintf("iteration %d", it+1), func(t *testing.T) {
					if err := compress.Document(doc, compress.Config{MinSimilarity: 0.8}); err != nil {
						t.Fatal(err)
					}

					if err := doc.Validate(); err != nil {
						t.Fatal(err)
					}

					doc.Components.SortMaps()

					gotDoc, err := doc.ToJSON()
					if err != nil {
						t.Fatal(err)
					}

					compareBytes(t, wantDoc, gotDoc)
				})
			}

			if tc.Name() == "toggl" {
				return
			}

			// Run enrich step again, check that nothing changes (no info got lost)
			iaData, err := testdata.ReadFile(filepath.Join("testdata", tc.Name(), "interactions.json"))
			if err != nil {
				t.Fatal(err)
			}

			interactions := cassette.Interactions{}
			if err := json.Unmarshal(iaData, &interactions, jsonOpts); err != nil {
				t.Fatal(err)
			}

			if err := enrich.Enrich(doc, interactions); err != nil {
				t.Fatal(err)
			}

			if err := doc.Validate(); err != nil {
				t.Fatal(err)
			}

			docAfterEnrich, err := doc.ToJSON()
			if err != nil {
				t.Fatal(err)
			}

			compareBytes(t, wantDoc, docAfterEnrich)
		})
	}
}

// compareBytes prints a compact diff of two byte slices
func compareBytes(t *testing.T, expected, actual []byte) {
	t.Helper()

	if bytes.Equal(expected, actual) {
		return
	}

	// Find first difference
	i := 0
	for i < len(expected) && i < len(actual) && expected[i] == actual[i] {
		i++
	}

	t.Errorf("\n┌─ Diff at offset %d\n│ Expected: %q\n│ Actual:   %q\n└─ %s",
		i, expected[i:min(len(expected), i+20)], actual[i:min(len(actual), i+20)],
		func() string {
			if len(expected) != len(actual) {
				return fmt.Sprintf("length %d vs %d", len(expected), len(actual))
			}
			return fmt.Sprintf("0x%02x vs 0x%02x", expected[i], actual[i])
		}())
}
