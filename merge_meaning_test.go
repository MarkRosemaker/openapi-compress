package compress_test

import (
	"strings"
	"testing"

	"github.com/MarkRosemaker/openapi"
	compress "github.com/MarkRosemaker/openapi-compress"
)

func TestDocument_DescriptionsStayWhereUsed(t *testing.T) {
	doc, err := openapi.LoadFromDataJSON([]byte(`{
  "openapi": "3.1.0",
  "info": {"title": "t", "version": "1"},
  "paths": {},
  "components": {"schemas": {
    "workspaceName": {"description": "The name of the bot's workspace.", "oneOf": [{"type": "string"}, {"type": "null"}]},
    "publicURL": {"description": "The public URL of the page.", "oneOf": [{"type": "string"}, {"type": "null"}]},
    "Bot": {"type": "object", "properties": {"name": {"$ref": "#/components/schemas/workspaceName"}}},
    "Page": {"type": "object", "properties": {
      "url": {"$ref": "#/components/schemas/publicURL"},
      "link": {"$ref": "#/components/schemas/publicURL", "description": "Where the page links to."}
    }}
  }}
}`))
	if err != nil {
		t.Fatal(err)
	}

	if err := compress.Document(doc, compress.Config{SkipNameShortening: true}); err != nil {
		t.Fatal(err)
	}

	if len(doc.Components.Schemas) != 3 {
		t.Fatalf("the two strings or null did not merge: %d schemas", len(doc.Components.Schemas))
	}

	bot, page := doc.Components.Schemas["Bot"].Properties, doc.Components.Schemas["Page"].Properties

	// each reference says what it said before, and the schema kept says nothing of either place
	for name, tc := range map[string]struct{ got, want string }{
		"Bot.name":  {bot["name"].Description, "The name of the bot's workspace."},
		"Page.url":  {page["url"].Description, "The public URL of the page."},
		"Page.link": {page["link"].Description, "Where the page links to."},
	} {
		if tc.got != tc.want {
			t.Errorf("%s: got %q, want %q", name, tc.got, tc.want)
		}
	}

	if d := bot["name"].Ref.Value.Description; d != "" {
		t.Errorf("the merged schema kept the description %q", d)
	}

	if err := doc.Validate(); err != nil {
		t.Fatal(err)
	}
}

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

func TestDocument_KeepsTheSpecificationsName(t *testing.T) {
	doc, err := openapi.LoadFromDataJSON([]byte(`{
  "openapi": "3.1.0",
  "info": {"title": "t", "version": "1"},
  "paths": {},
  "components": {"schemas": {
    "PageParentID": {"type": "object", "properties": {"id": {"type": "string"}}, "required": ["id"],
      "x-flattened-from": "#/components/schemas/Page/properties/parent"},
    "idObjectResponse": {"type": "object", "properties": {"id": {"type": "string"}}, "required": ["id"]},
    "Page": {"type": "object", "x-go-name": "Page", "properties": {
      "parent": {"$ref": "#/components/schemas/PageParentID"},
      "owner": {"$ref": "#/components/schemas/PageParentID"},
      "space": {"$ref": "#/components/schemas/idObjectResponse"}
    }}
  }}
}`))
	if err != nil {
		t.Fatal(err)
	}

	if err := compress.Document(doc, compress.Config{SkipNameShortening: true}); err != nil {
		t.Fatal(err)
	}

	// a name the specification gave wins over one flatten made up, however often that one is referred to
	if _, ok := doc.Components.Schemas["idObjectResponse"]; !ok {
		t.Error("idObjectResponse was merged away")
	}

	if _, ok := doc.Components.Schemas["PageParentID"]; ok {
		t.Error("PageParentID was kept")
	}

	// the mark is gone, and other extensions stay
	for name, s := range doc.Components.Schemas {
		if strings.Contains(string(s.Extensions), "x-flattened-from") {
			t.Errorf("%s still has x-flattened-from", name)
		}
	}

	if got := string(doc.Components.Schemas["Page"].Extensions); got != `{"x-go-name":"Page"}` {
		t.Errorf("Page's extensions are %s", got)
	}
}

func TestDocument_KeepsExamplesOfAlternatives(t *testing.T) {
	doc, err := openapi.LoadFromDataJSON([]byte(`{
  "openapi": "3.1.0",
  "info": {"title": "t", "version": "1"},
  "paths": {},
  "components": {"schemas": {
    "aName": {"oneOf": [{"type": "string"}, {"type": "null"}]},
    "bURL": {"oneOf": [{"type": "string", "example": "https://example.com"}, {"type": "null"}]},
    "Page": {"type": "object", "properties": {
      "name": {"$ref": "#/components/schemas/aName"},
      "title": {"$ref": "#/components/schemas/aName"},
      "url": {"$ref": "#/components/schemas/bURL"}
    }}
  }}
}`))
	if err != nil {
		t.Fatal(err)
	}

	if err := compress.Document(doc, compress.Config{SkipNameShortening: true}); err != nil {
		t.Fatal(err)
	}

	// aName is kept, for its more references, and gets the example bURL had for its string
	kept := doc.Components.Schemas["aName"]
	if got := string(kept.OneOf[0].Example); got != `"https://example.com"` {
		t.Errorf("the string alternative's example is %s, want the merged schema's", got)
	}
}
