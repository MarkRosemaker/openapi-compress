package compress_test

import (
	"testing"

	"github.com/MarkRosemaker/openapi"
	compress "github.com/MarkRosemaker/openapi-compress"
)

// TestDocument_ParametersKeepTheirGoNames: two parameters of one wire name, such as IBKR Flex's q, which is a query's ID
// in one request and a reference code in another, stay two when their x-go-name tells them apart; a parameter the same
// as another in every way is folded into it.
func TestDocument_ParametersKeepTheirGoNames(t *testing.T) {
	t.Parallel()

	doc, err := openapi.LoadFromDataJSON([]byte(`{
  "openapi": "3.1.0",
  "info": {"title": "t", "version": "1"},
  "paths": {
    "/SendRequest": {"get": {
      "parameters": [{"$ref": "#/components/parameters/QueryID"}],
      "responses": {"200": {"description": "OK"}}
    }},
    "/GetStatement": {"get": {
      "parameters": [{"$ref": "#/components/parameters/ReferenceCode"}, {"$ref": "#/components/parameters/Version2"}],
      "responses": {"200": {"description": "OK"}}
    }}
  },
  "components": {"parameters": {
    "QueryID": {"name": "q", "in": "query", "schema": {"type": "string"}, "x-go-name": "QueryID"},
    "ReferenceCode": {"name": "q", "in": "query", "schema": {"type": "string"}, "x-go-name": "ReferenceCode"},
    "Version": {"name": "v", "in": "query", "schema": {"type": "integer"}, "x-go-name": "Version"},
    "Version2": {"name": "v", "in": "query", "schema": {"type": "integer"}, "x-go-name": "Version"}
  }}
}`))
	if err != nil {
		t.Fatal(err)
	}

	if err := compress.Document(doc, compress.Config{}); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"QueryID", "ReferenceCode", "Version"} {
		if _, ok := doc.Components.Parameters[name]; !ok {
			t.Errorf("parameter %s was folded away", name)
		}
	}

	if _, ok := doc.Components.Parameters["Version2"]; ok {
		t.Error("Version2, the same as Version, was kept")
	}

	if got := doc.Paths["/GetStatement"].Get.Parameters[1].Ref.Identifier; got != "#/components/parameters/Version" {
		t.Errorf("GetStatement refers to %s, want Version", got)
	}
}
