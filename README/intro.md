Specifications that are generated rather than hand-written accumulate duplicates
fast. The same object appears as a response body, as an array element, and as a
nested property, and each occurrence gets its own component with its own unwieldy
name — `GetV1PetByPetIDOkJSONResponseMedicalInfo` and
`ListV1PetsOkJSONResponseDataItemsMedicalInfo` describing an identical shape.

This module collapses them back down. Two schemas are considered the same when the
same JSON validates against both, using
[`openapi-compare`](https://github.com/MarkRosemaker/openapi-compare) — so schemas
that differ only in their `title` or `description` still merge, while a difference
that changes what the schema accepts prevents it.

Optionally it goes further, merging schemas that are merely *similar*: if two object
schemas share most of their properties, they can be widened into a single schema
covering both.
