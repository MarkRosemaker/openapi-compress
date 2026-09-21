package compress_test

import (
	"encoding/json/jsontext"
	"testing"

	"github.com/MarkRosemaker/openapi"
	compress "github.com/MarkRosemaker/openapi-compress"
)

func TestDocument_TrimExamples(t *testing.T) {
	t.Parallel()

	s := &openapi.Schema{
		Type:    openapi.TypeInteger,
		Example: jsontext.Value(`[1,2,3,4,5]`),
	}

	d := &openapi.Document{
		OpenAPI: "3.1.0",
		Info:    &openapi.Info{Title: "test", Version: "0.0.0"},
	}
	d.Components.Schemas = openapi.Schemas{}
	d.Components.Schemas.Set("Count", s)

	if err := compress.Document(d, compress.Config{TrimExamples: 2}); err != nil {
		t.Fatal(err)
	}

	if got, want := string(s.Example), `[1,2]`; got != want {
		t.Errorf("Example = %s, want %s", got, want)
	}
}

func TestDocument_TrimExamplesDisabledByDefault(t *testing.T) {
	t.Parallel()

	s := &openapi.Schema{
		Type:    openapi.TypeInteger,
		Example: jsontext.Value(`[1,2,3,4,5]`),
	}

	d := &openapi.Document{
		OpenAPI: "3.1.0",
		Info:    &openapi.Info{Title: "test", Version: "0.0.0"},
	}
	d.Components.Schemas = openapi.Schemas{}
	d.Components.Schemas.Set("Count", s)

	if err := compress.Document(d, compress.Config{}); err != nil {
		t.Fatal(err)
	}

	if got, want := string(s.Example), `[1,2,3,4,5]`; got != want {
		t.Errorf("Example = %s, want %s (should be untouched)", got, want)
	}
}
