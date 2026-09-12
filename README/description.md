---
tagline: One type, one name, one definition.
logo:
    alt: A gopher pressing a tall stack of identical boxes down into a single box
    source: openapi-compress.jpg
    width: 500
---

<div align="center" id=badges>

![Code Coverage](https://img.shields.io/badge/coverage-76.2%25-green)

</div>





`openapi-compress` removes redundancy from an
[OpenAPI 3.x](https://spec.openapis.org/oas/v3.1.0) specification. It finds
component schemas that describe the same thing, merges them into one, rewrites every
`$ref` that pointed at the copies, and shortens the resulting names.
