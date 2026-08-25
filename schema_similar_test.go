package compress

import (
	"testing"

	"github.com/MarkRosemaker/openapi"
)

// inlineRef creates a SchemaRef wrapping an inline (non-$ref) schema.
func inlineRef(s *openapi.Schema) *openapi.SchemaRef {
	return &openapi.SchemaRef{Value: s}
}

// stringSchema returns a minimal string schema.
func stringSchema() *openapi.Schema {
	return &openapi.Schema{Type: openapi.TypeString}
}

// // intSchema returns a minimal integer schema.
// func intSchema() *openapi.Schema {
// 	return &openapi.Schema{Type: openapi.TypeInteger}
// }

// objectSchema builds an object schema with the given required properties.
// All listed properties are set to inline string schemas and marked required.
func objectSchema(required []string, optional ...string) *openapi.Schema {
	s := &openapi.Schema{
		Type:     openapi.TypeObject,
		Required: append([]string(nil), required...),
	}
	for _, name := range required {
		s.Properties.Set(name, inlineRef(stringSchema()))
	}
	for _, name := range optional {
		s.Properties.Set(name, inlineRef(stringSchema()))
	}
	return s
}

// ─── schemasSimilarity ───────────────────────────────────────────────────────

func TestSchemasSimilarity_Equal(t *testing.T) {
	s := objectSchema([]string{"a", "b"})
	if got := schemasSimilarity(s, s); got != 1.0 {
		t.Errorf("same pointer: want 1.0, got %v", got)
	}
}

func TestSchemasSimilarity_StructurallyEqual(t *testing.T) {
	a := objectSchema([]string{"a", "b"})
	b := objectSchema([]string{"a", "b"})
	if got := schemasSimilarity(a, b); got != 1.0 {
		t.Errorf("structurally equal: want 1.0, got %v", got)
	}
}

func TestSchemasSimilarity_DifferentTypes(t *testing.T) {
	a := &openapi.Schema{Type: openapi.TypeString}
	b := &openapi.Schema{Type: openapi.TypeInteger}
	if got := schemasSimilarity(a, b); got != 0.0 {
		t.Errorf("different types: want 0.0, got %v", got)
	}
}

func TestSchemasSimilarity_NonObject(t *testing.T) {
	// Non-object schemas that are not structurally equal return 0.
	a := &openapi.Schema{Type: openapi.TypeString, Format: "email"}
	b := &openapi.Schema{Type: openapi.TypeString, Format: "uri"}
	if got := schemasSimilarity(a, b); got != 0.0 {
		t.Errorf("non-object string schemas: want 0.0, got %v", got)
	}
}

func TestSchemasSimilarity_FullOverlap(t *testing.T) {
	// Symmetric difference is empty → all 3 shared props in union of 3 → score = 3/3 = 1.0.
	a := objectSchema([]string{"x", "y", "z"})
	b := objectSchema([]string{"x", "y", "z"})
	if got := schemasSimilarity(a, b); got != 1.0 {
		t.Errorf("full overlap: want 1.0, got %v", got)
	}
}

func TestSchemasSimilarity_HalfOverlap(t *testing.T) {
	// a has {x, y}, b has {y, z}: union={x,y,z} (3), y matches → score=1/3 ≈ 0.333.
	a := objectSchema([]string{"x", "y"})
	b := objectSchema([]string{"y", "z"})
	got := schemasSimilarity(a, b)
	want := 1.0 / 3.0
	if abs(got-want) > 1e-9 {
		t.Errorf("half overlap: want %v, got %v", want, got)
	}
}

func TestSchemasSimilarity_LowOverlap(t *testing.T) {
	// a has {a,b,c,d,e}, b has {e,f,g,h,i}: union=9, one match → score=1/9.
	a := objectSchema([]string{"a", "b", "c", "d", "e"})
	b := objectSchema([]string{"e", "f", "g", "h", "i"})
	got := schemasSimilarity(a, b)
	want := 1.0 / 9.0
	if abs(got-want) > 1e-9 {
		t.Errorf("low overlap: want %v, got %v", want, got)
	}
}

func TestSchemasSimilarity_PartialCredit(t *testing.T) {
	// Same property name, same type, different detail → 0.5 credit.
	// union={x}: score = 0.5/1 = 0.5.
	a := &openapi.Schema{
		Type: openapi.TypeObject,
		Properties: func() openapi.SchemaRefs {
			var refs openapi.SchemaRefs
			refs.Set("x", inlineRef(&openapi.Schema{Type: openapi.TypeInteger}))
			return refs
		}(),
	}
	b := &openapi.Schema{
		Type: openapi.TypeObject,
		Properties: func() openapi.SchemaRefs {
			var refs openapi.SchemaRefs
			refs.Set("x", inlineRef(&openapi.Schema{Type: openapi.TypeNumber}))
			return refs
		}(),
	}
	got := schemasSimilarity(a, b)
	if abs(got-0.5) > 1e-9 {
		t.Errorf("partial credit: want 0.5, got %v", got)
	}
}

func TestSchemasSimilarity_IrreconcilableProperty(t *testing.T) {
	// A property holding an array on one side and an object on the other
	// cannot be widened to cover both: merging keeps a's, leaving the result
	// claiming a shape b's data does not have. No agreement elsewhere makes
	// the two mergeable, so the score is 0 however many properties match.
	props := func(x *openapi.Schema) openapi.SchemaRefs {
		var refs openapi.SchemaRefs
		refs.Set("same1", inlineRef(&openapi.Schema{Type: openapi.TypeBoolean}))
		refs.Set("x", inlineRef(x))
		refs.Set("same2", inlineRef(&openapi.Schema{Type: openapi.TypeString}))
		return refs
	}

	a := &openapi.Schema{Type: openapi.TypeObject, Properties: props(&openapi.Schema{Type: openapi.TypeArray})}
	b := &openapi.Schema{Type: openapi.TypeObject, Properties: props(&openapi.Schema{Type: openapi.TypeObject})}

	if got := schemasSimilarity(a, b); got != 0.0 {
		t.Errorf("irreconcilable property: want 0, got %v", got)
	}
}

// ─── mergeSchemas ────────────────────────────────────────────────────────────

// TestMergeSchemas_RequiredIntersection checks the core contract: properties
// that are required in both schemas remain required; properties unique to one
// schema become optional in the merged result.
func TestMergeSchemas_RequiredIntersection(t *testing.T) {
	// a requires {shared, onlyA}, b requires {shared, onlyB}.
	// After merge into a: required = {shared}; onlyA and onlyB both present but not required.
	a := objectSchema([]string{"shared", "onlyA"})
	b := objectSchema([]string{"shared", "onlyB"})

	mergeSchemas(a, b)

	// Required must be exactly {shared}.
	if len(a.Required) != 1 || a.Required[0] != "shared" {
		t.Errorf("required after merge: want [shared], got %v", a.Required)
	}

	// All three properties must be present.
	for _, name := range []string{"shared", "onlyA", "onlyB"} {
		if _, ok := a.Properties[name]; !ok {
			t.Errorf("property %q missing after merge", name)
		}
	}
}

func TestMergeSchemas_AllRequiredBecomesNone(t *testing.T) {
	// Disjoint required sets → required intersection is empty after merge.
	a := objectSchema([]string{"a1", "a2"})
	b := objectSchema([]string{"b1", "b2"})

	mergeSchemas(a, b)

	if len(a.Required) != 0 {
		t.Errorf("required after merge: want [], got %v", a.Required)
	}

	for _, name := range []string{"a1", "a2", "b1", "b2"} {
		if _, ok := a.Properties[name]; !ok {
			t.Errorf("property %q missing after merge", name)
		}
	}
}

func TestMergeSchemas_UnionOfProperties(t *testing.T) {
	// a has {p, q}, b has {q, r, s}. After merge: a has {p, q, r, s}.
	a := objectSchema([]string{"p", "q"})
	b := objectSchema([]string{"q", "r", "s"})

	mergeSchemas(a, b)

	for _, name := range []string{"p", "q", "r", "s"} {
		if _, ok := a.Properties[name]; !ok {
			t.Errorf("property %q missing after merge", name)
		}
	}
	if len(a.Properties) != 4 {
		t.Errorf("want 4 properties after merge, got %d", len(a.Properties))
	}
}

func TestMergeSchemas_IntegerPlusNumberBecomesNumber(t *testing.T) {
	// Property "val": integer in a, number in b → result should be number.
	a := &openapi.Schema{
		Type:     openapi.TypeObject,
		Required: []string{"val"},
		Properties: func() openapi.SchemaRefs {
			var refs openapi.SchemaRefs
			refs.Set("val", inlineRef(&openapi.Schema{Type: openapi.TypeInteger}))
			return refs
		}(),
	}
	b := &openapi.Schema{
		Type:     openapi.TypeObject,
		Required: []string{"val"},
		Properties: func() openapi.SchemaRefs {
			var refs openapi.SchemaRefs
			refs.Set("val", inlineRef(&openapi.Schema{Type: openapi.TypeNumber}))
			return refs
		}(),
	}

	mergeSchemas(a, b)

	ref, ok := a.Properties["val"]
	if !ok {
		t.Fatal("property 'val' missing after merge")
	}
	if ref.Value == nil {
		t.Fatal("expected inline schema for 'val', got nil value")
	}
	if ref.Value.Type != openapi.TypeNumber {
		t.Errorf("'val' type after merge: want number, got %v", ref.Value.Type)
	}
}

// ─── Document integration ────────────────────────────────────────────────────

// TestDocument_DifferentTypesNeverMerge verifies that schemas of different
// types are never merged regardless of how low MinSimilarity is set.
func TestDocument_DifferentTypesNeverMerge(t *testing.T) {
	d := minimalDocument(map[string]*openapi.Schema{
		"A": {Type: openapi.TypeString},
		"B": {Type: openapi.TypeInteger},
	})

	if err := Document(d, Config{MinSimilarity: 0.0}); err != nil {
		t.Fatal(err)
	}

	if len(d.Components.Schemas) != 2 {
		t.Errorf("want 2 schemas (no merge), got %d", len(d.Components.Schemas))
	}
}

// TestDocument_ThresholdDependent checks that two similar-but-not-equal
// schemas merge at a permissive threshold but not at a strict one.
//
// Schema A: {shared, onlyA} all required  → sim with B = 1/3 ≈ 0.333
// Schema B: {shared, onlyB} all required
func TestDocument_ThresholdDependent(t *testing.T) {
	makeDoc := func() *openapi.Document {
		return minimalDocument(map[string]*openapi.Schema{
			"A": objectSchema([]string{"shared", "onlyA"}),
			"B": objectSchema([]string{"shared", "onlyB"}),
		})
	}

	// At MinSimilarity=0.3: sim≈0.333 >= 0.3 → should merge.
	d := makeDoc()
	if err := Document(d, Config{MinSimilarity: 0.3, SimilarityStep: 0.1}); err != nil {
		t.Fatal(err)
	}
	if len(d.Components.Schemas) != 1 {
		t.Errorf("threshold 0.3: want 1 schema after merge, got %d", len(d.Components.Schemas))
	}

	// At MinSimilarity=0.5: sim≈0.333 < 0.5 → should NOT merge.
	d = makeDoc()
	if err := Document(d, Config{MinSimilarity: 0.5, SimilarityStep: 0.1}); err != nil {
		t.Fatal(err)
	}
	if len(d.Components.Schemas) != 2 {
		t.Errorf("threshold 0.5: want 2 schemas (no merge), got %d", len(d.Components.Schemas))
	}
}

// TestDocument_ExamplesIgnoredForEquality confirms that two schemas that are
// structurally identical but have different example values are still merged
// at threshold=1.0.
func TestDocument_ExamplesIgnoredForEquality(t *testing.T) {
	// Build two schemas that are identical except for their Example field.
	schemaA := &openapi.Schema{Type: openapi.TypeString}
	schemaA.Example = []byte(`"alice"`)

	schemaB := &openapi.Schema{Type: openapi.TypeString}
	schemaB.Example = []byte(`"bob"`)

	d := minimalDocument(map[string]*openapi.Schema{
		"A": schemaA,
		"B": schemaB,
	})

	if err := Document(d, Config{}); err != nil {
		t.Fatal(err)
	}

	if len(d.Components.Schemas) != 1 {
		t.Errorf("want 1 schema after merging identical schemas with different examples, got %d", len(d.Components.Schemas))
	}
}

// TestDocument_NonObjectSameShapeDifferentDescriptionMerges confirms that two
// non-object schemas with the same shape but different documentation (title,
// description) are merged at threshold=1.0. Before switching to
// schema.SameShape, schemasSimilarity returned 0.0 for any non-object schemas
// that weren't byte-for-byte equal, so schemas like this - identical in every
// way that affects validation, differing only in description - could never be
// deduplicated.
func TestDocument_NonObjectSameShapeDifferentDescriptionMerges(t *testing.T) {
	schemaA := &openapi.Schema{Type: openapi.TypeString, Format: "email", Description: "the user's email"}
	schemaB := &openapi.Schema{Type: openapi.TypeString, Format: "email", Description: "email address"}

	d := minimalDocument(map[string]*openapi.Schema{
		"A": schemaA,
		"B": schemaB,
	})

	if err := Document(d, Config{}); err != nil {
		t.Fatal(err)
	}

	if len(d.Components.Schemas) != 1 {
		t.Errorf("want 1 schema after merging same-shape schemas with different descriptions, got %d", len(d.Components.Schemas))
	}
}

// ─── splitCamelCase ──────────────────────────────────────────────────────────

func TestSplitCamelCase(t *testing.T) {
	cases := []struct {
		in   string
		want []string
	}{
		{"GetV1PetByPetID", []string{"Get", "V1", "Pet", "By", "Pet", "ID"}},
		{"OkJSONResponse", []string{"Ok", "JSON", "Response"}},
		{"AccountsPayableCurrent", []string{"Accounts", "Payable", "Current"}},
		// digit → upper-letter boundary
		{"Cik0000320193JSON", []string{"Cik0000320193", "JSON"}},
		{
			"ListAPIXbrlCompanyfactsCik0000320193JSONOkResponse",
			[]string{"List", "API", "Xbrl", "Companyfacts", "Cik0000320193", "JSON", "Ok", "Response"},
		},
	}
	for _, tc := range cases {
		got := splitCamelCase(tc.in)
		if len(got) != len(tc.want) {
			t.Errorf("splitCamelCase(%q): got %v, want %v", tc.in, got, tc.want)
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("splitCamelCase(%q)[%d]: got %q, want %q (full: %v)", tc.in, i, got[i], tc.want[i], got)
			}
		}
	}
}

// ─── shortName ───────────────────────────────────────────────────────────────

func TestShortName_RemovesNoise(t *testing.T) {
	schemas := make(openapi.Schemas)
	got := shortName("GetV1PetByPetIDOkJSONResponseMedicalInfo", schemas)
	want := "PetByPetIDMedicalInfo"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestShortName_StripsCIKNumber(t *testing.T) {
	// Digit-only tokens (like embedded CIK numbers) are stripped.
	schemas := make(openapi.Schemas)
	got := shortName("ListAPIXbrlCompanyfactsCik0000320193JSONOkResponse", schemas)
	// "List","API","Xbrl","Companyfacts","Cik0000320193","JSON","Ok","Response"
	// noise removed: "JSON","Ok","Response"
	// digit stripped: "Cik0000320193" → "Cik"
	want := "ListAPIXbrlCompanyfactsCik"
	if got != want {
		t.Errorf("want %q, got %q", want, got)
	}
}

func TestShortName_Unchanged(t *testing.T) {
	// A name that only consists of meaningful words → no change.
	schemas := make(openapi.Schemas)
	got := shortName("MedicalInfo", schemas)
	if got != "MedicalInfo" {
		t.Errorf("want unchanged %q, got %q", "MedicalInfo", got)
	}
}

func TestShortName_UniqueSuffix(t *testing.T) {
	// When the shortened name collides, a numeric suffix is appended.
	schemas := make(openapi.Schemas)
	schemas.Set("PetByPetID", &openapi.Schema{Type: openapi.TypeObject})

	got := shortName("GetV1PetByPetIDOkJSONResponse", schemas)
	if got != "PetByPetID2" {
		t.Errorf("want %q, got %q", "PetByPetID2", got)
	}
}

// TestDocument_ShortensMergedCanonicals verifies that after compression the
// merged canonical schema gets a shorter name while non-merged schemas keep
// their original names.
func TestDocument_ShortensMergedCanonicals(t *testing.T) {
	d := minimalDocument(map[string]*openapi.Schema{
		"GetV1PetByPetIDOkJSONResponseMedicalInfo":     {Type: openapi.TypeString},
		"ListV1PetsOkJSONResponseDataItemsMedicalInfo": {Type: openapi.TypeString},
		"GetV1PetByPetIDOkJSONResponseBreed":           {Type: openapi.TypeObject},
	})

	if err := Document(d, Config{}); err != nil {
		t.Fatal(err)
	}

	// The two identical string schemas merged; the canonical (lex-smaller) got
	// shortened: "GetV1PetByPetIDOkJSONResponseMedicalInfo" → "PetByPetIDMedicalInfo".
	// The unique object schema was never merged so keeps its name.
	if _, ok := d.Components.Schemas["PetByPetIDMedicalInfo"]; !ok {
		t.Errorf("expected shortened name 'PetByPetIDMedicalInfo', got schemas: %v", schemaKeys(d))
	}
	if _, ok := d.Components.Schemas["GetV1PetByPetIDOkJSONResponseBreed"]; !ok {
		t.Error("non-merged schema should keep its original name")
	}
}

func schemaKeys(d *openapi.Document) []string {
	keys := make([]string, 0, len(d.Components.Schemas))
	for k := range d.Components.Schemas {
		keys = append(keys, k)
	}
	return keys
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// minimalDocument builds a bare-minimum *openapi.Document with the given schemas.
func minimalDocument(schemas map[string]*openapi.Schema) *openapi.Document {
	d := &openapi.Document{
		OpenAPI: "3.1.0",
		Info:    &openapi.Info{Title: "test", Version: "0.0.0"},
	}
	for name, s := range schemas {
		d.Components.Schemas.Set(name, s)
	}
	return d
}
