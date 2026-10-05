- **Deduplicates identical component schemas**, keeping one canonical definition
- **Merges similar schemas** above a configurable similarity threshold
- **Rewrites every `$ref`** throughout the document, including deep inside nested schemas
- **Deduplicates parameters** in the same way
- **Shortens the names** of merged schemas, dropping generated noise like `OkJSONResponse`

### How similarity works

At the default threshold of `1.0` only equivalent schemas merge. Below that, object
schemas are scored by a weighted Jaccard index over their property names:

```
score = Σ weight(p) / |union of property names|
```

where a property present in both with the same shape scores `1.0`, and one present
in both with a different shape scores `0.5`. The threshold steps down gradually,
running each level until no further merges are found, so the most confident merges
always happen first.

A bare scalar is never merged: a string, number, integer or boolean with nothing
but a type and documentation, such as a component `idRequest` that is only
`{"type": "string"}`. Its name and description are all it carries, and merging by
shape would erase exactly those, giving IDs, emoji names and time zones one type.
A scalar with a constraint, such as a `format` or an `enum`, merges like any other
schema.

### What a merge keeps

Of the schemas that merge, the one kept is one the specification named, else the
one with the most references, else the one with the shortest name, else the
alphabetically first: the `{id}` object a specification defines keeps its own
name rather than taking one derived from a single place that also used it. A
schema `openapi-flatten` moved out of the document and named itself carries
`x-flattened-from`, given its `MarkOrigin`; compress reads and removes it before
comparing anything, so it never keeps two schemas apart.

Examples are kept from every schema that merges, down to each property, item and
alternative: where the one kept has none, it takes one from those merged into it.

A description says what a schema is used for in one place, not what shape it
has. When the schemas that merge disagree on it, each one's description moves
beside the `$ref`s that pointed at it, and the schema kept has none, so no
reference shows another's. A reference with a description of its own keeps it.

