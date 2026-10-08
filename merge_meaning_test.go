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
