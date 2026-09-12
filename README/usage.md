```bash
go get -tool github.com/MarkRosemaker/openapi-compress/cmd/openapi-compress
```

or

```bash
go get github.com/MarkRosemaker/openapi-compress
```


```go
import (
    "github.com/MarkRosemaker/openapi"
    compress "github.com/MarkRosemaker/openapi-compress"
)

doc, err := openapi.LoadFromFile("api/openapi.json")
if err != nil {
    log.Fatal(err)
}

// Exact-shape deduplication only.
if err := compress.Document(doc, compress.Config{}); err != nil {
    log.Fatal(err)
}
```

To also merge schemas that merely overlap, lower the threshold:

```go
err := compress.Document(doc, compress.Config{
    MinSimilarity:  0.8,  // merge schemas sharing ≥80% of their properties
    SimilarityStep: 0.05, // step down by this much per round
})
```

| Field | Default | Purpose |
|---|---|---|
| `MinSimilarity` | `1.0` | Lowest similarity at which two schemas may merge. `1.0` means equivalent shapes only. |
| `SimilarityStep` | `0.05` | How far the threshold drops between rounds. |
| `SkipNameShortening` | `false` | Keep the original names instead of shortening merged ones. |

Renaming a schema on its own — without compressing anything — is
[`openapi-edit`](https://github.com/MarkRosemaker/openapi-edit)'s job, and this
module uses it internally to shorten the names of merged schemas:

```go
err := edit.RenameSchema(doc, "OldName", "NewName")
```

### Command line

```sh
openapi-compress -spec api/openapi.json -minsim 0.8
```

| Flag | Default | Purpose |
|---|---|---|
| `-spec` | `api/openapi.json` | Path to the specification, rewritten in place |
| `-minsim` | `1` | Minimum similarity for a merge |
| `-simstep` | `0.05` | Threshold reduction between rounds |

If the specification was valid on the way in, the result is validated before it is
written back.
