package compress_test

import (
	"slices"
	"testing"

	"github.com/MarkRosemaker/openapi"
	compress "github.com/MarkRosemaker/openapi-compress"
)

func TestDocument_KeepsBareScalars(t *testing.T) {
	doc, err := openapi.LoadFromDataJSON([]byte(`{
  "openapi": "3.1.0",
  "info": {"title": "t", "version": "1"},
  "paths": {},
  "components": {"schemas": {
    "emojiRequest": {"type": "string", "example": "😀"},
    "idRequest": {"type": "string", "description": "An ID."},
    "createdAt": {"type": "string", "format": "date-time"},
    "updatedAt": {"type": "string", "format": "date-time"},
    "Page": {"type": "object", "properties": {
      "id": {"$ref": "#/components/schemas/idRequest"},
      "icon": {"$ref": "#/components/schemas/emojiRequest"},
      "created": {"$ref": "#/components/schemas/createdAt"},
      "updated": {"$ref": "#/components/schemas/updatedAt"}
    }}
  }}
}`))
	if err != nil {
		t.Fatal(err)
	}

	if err := compress.Document(doc, compress.Config{MinSimilarity: 0.8}); err != nil {
		t.Fatal(err)
	}

	var names []string
	for name := range doc.Components.Schemas {
		names = append(names, name)
	}

	slices.Sort(names)

	// a bare string keeps its name and documentation; one with a format still merges with its equal
	if !slices.Contains(names, "emojiRequest") || !slices.Contains(names, "idRequest") {
		t.Errorf("bare strings were merged: got %v", names)
	}

	if slices.Contains(names, "createdAt") && slices.Contains(names, "updatedAt") {
		t.Errorf("strings with the same format were not merged: got %v", names)
	}

	if ex := doc.Components.Schemas["idRequest"].Example; ex != nil {
		t.Errorf("idRequest got the example %s", ex)
	}
}
