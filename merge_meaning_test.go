package compress_test

import (
	"testing"

	"github.com/MarkRosemaker/openapi"
	compress "github.com/MarkRosemaker/openapi-compress"
)

func TestDocument_KeepsTheMostReferencedName(t *testing.T) {
	doc, err := openapi.LoadFromDataJSON([]byte(`{
  "openapi": "3.1.0",
  "info": {"title": "t", "version": "1"},
  "paths": {},
  "components": {"schemas": {
    "FileUploadPageCoverFileUpload": {"type": "object", "properties": {"id": {"type": "string"}}, "required": ["id"]},
    "idObject": {"type": "object", "properties": {"id": {"type": "string"}}, "required": ["id"]},
    "Cover": {"type": "object", "properties": {"file_upload": {"$ref": "#/components/schemas/FileUploadPageCoverFileUpload"}}},
    "Page": {"type": "object", "properties": {
      "parent": {"$ref": "#/components/schemas/idObject"},
      "owner": {"$ref": "#/components/schemas/idObject"}
    }}
  }}
}`))
	if err != nil {
		t.Fatal(err)
	}

	if err := compress.Document(doc, compress.Config{SkipNameShortening: true}); err != nil {
		t.Fatal(err)
	}

	// the alphabetically first name loses to the one referenced more
	if _, ok := doc.Components.Schemas["idObject"]; !ok {
		t.Error("idObject was merged away")
	}

	if _, ok := doc.Components.Schemas["FileUploadPageCoverFileUpload"]; ok {
		t.Error("FileUploadPageCoverFileUpload was kept")
	}
}

// TestDocument_EnumsKeepTheirMeaning: two enums the specification named, of the same values but described apart, are
// two types; an enum flatten named from an inline one merges with a named enum of the same values.
func TestDocument_EnumsKeepTheirMeaning(t *testing.T) {
	doc, err := openapi.LoadFromDataJSON([]byte(`{
  "openapi": "3.1.0",
  "info": {"title": "t", "version": "1"},
  "paths": {},
  "components": {"schemas": {
    "NodeType": {"description": "1 Folder, 2 File, 3 Album", "type": "integer", "enum": [1, 2, 3]},
    "ThumbnailType": {"description": "1 Preview, 2 HD preview, 3 machine learning", "type": "integer", "enum": [1, 2, 3]},
    "RespKind": {"description": "The kind of the response", "type": "integer", "enum": [1, 2, 3], "x-flattened-from": "Resp.kind"},
    "Resp": {"type": "object", "properties": {
      "Type": {"$ref": "#/components/schemas/NodeType"},
      "Thumb": {"$ref": "#/components/schemas/ThumbnailType"},
      "kind": {"$ref": "#/components/schemas/RespKind"}
    }}
  }}
}`))
	if err != nil {
		t.Fatal(err)
	}

	if err := compress.Document(doc, compress.Config{SkipNameShortening: true}); err != nil {
		t.Fatal(err)
	}

	props := doc.Components.Schemas["Resp"].Properties
	if got := props["Thumb"].Ref.Identifier; got != "#/components/schemas/ThumbnailType" {
		t.Errorf("Thumb is %s, want ThumbnailType", got)
	}

	if got := props["Type"].Ref.Identifier; got != "#/components/schemas/NodeType" {
		t.Errorf("Type is %s, want NodeType", got)
	}

	if _, ok := doc.Components.Schemas["RespKind"]; ok {
		t.Error("RespKind, which flatten named, was kept beside an enum of the same values")
	}
}

// TestDocument_DefaultsStayWhereUsed: two enums flatten named, alike but for their defaults, merge, and each default
// stays beside the references to its enum, since what a value is when left out is the request's.
func TestDocument_DefaultsStayWhereUsed(t *testing.T) {
	doc, err := openapi.LoadFromDataJSON([]byte(`{
  "openapi": "3.1.0",
  "info": {"title": "t", "version": "1"},
  "paths": {},
  "components": {"schemas": {
    "MapView": {"type": "string", "enum": ["low", "high"], "default": "high", "x-flattened-from": "Map.view"},
    "RoomView": {"type": "string", "enum": ["low", "high"], "default": "low", "x-flattened-from": "Room.view"},
    "Map": {"type": "object", "properties": {"view": {"$ref": "#/components/schemas/MapView"}, "width": {"type": "integer"}}},
    "Room": {"type": "object", "properties": {"view": {"$ref": "#/components/schemas/RoomView"}}}
  }}
}`))
	if err != nil {
		t.Fatal(err)
	}

	if err := compress.Document(doc, compress.Config{SkipNameShortening: true}); err != nil {
		t.Fatal(err)
	}

	schemas := doc.Components.Schemas
	if len(schemas) != 3 {
		t.Fatalf("the views did not merge: %d schemas", len(schemas))
	}

	for owner, want := range map[string]string{"Map": `"high"`, "Room": `"low"`} {
		if got := string(schemas[owner].Properties["view"].Default); got != want {
			t.Errorf("%s.view defaults to %s, want %s", owner, got, want)
		}
	}

	for name, s := range schemas {
		if len(s.Enum) > 0 && len(s.Default) > 0 {
			t.Errorf("%s, the enum kept, has a default of its own: %s", name, s.Default)
		}
	}
}
