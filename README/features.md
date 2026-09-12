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
