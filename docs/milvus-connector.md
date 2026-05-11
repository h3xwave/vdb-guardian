# Milvus Connector

The minimal Milvus connector implements the shared `connectors.Connector` interface for source-side retrieval behavior collection in the Milvus to pgvector migration MVP.

## Current scope

Implemented capabilities:

- Validate Milvus connector configuration.
- Create a real Milvus Go SDK adapter from a Milvus address.
- Connect, count, search, and close through a small adapter boundary.
- Read Milvus collection `row_count` through SDK collection statistics.
- Execute one-query vector search through the SDK and convert Milvus hits into normalized `connectors.SearchResponse` values.
- Normalize metric score direction so larger `SearchHit.Score` values are better.

Not yet implemented:

- Collection creation from the connector package.
- Index creation or load orchestration from the connector package.
- Fixture seeding into Milvus through a CLI.
- Metadata filters or Milvus boolean expressions.
- Integration tests against the Docker migration stack.

The SDK adapter is intentionally narrow and remains behind an internal adapter boundary so connector normalization and SDK request/result conversion stay unit-testable without Docker or network state.

## Configuration

The connector is configured with `MilvusConfig`:

```go
type MilvusConfig struct {
    Name              string
    Address           string
    DefaultCollection string
    IDField           string
    VectorField       string
    Metric            string
}
```

Defaults:

```text
Name:              milvus
DefaultCollection: items
IDField:           id
VectorField:       embedding
Metric:            cosine
```

The local Docker stack exposes Milvus at `localhost:19530` by default, but production or shared environment addresses must be supplied through runtime configuration. Do not commit real credentials or private endpoints.

## Search behavior

The connector maps the shared `SearchRequest` into a Milvus adapter request:

```text
SearchRequest.Collection  -> collection name, or DefaultCollection when empty
SearchRequest.QueryVector -> query vector
SearchRequest.ExpandK     -> Milvus search limit
SearchRequest.Params      -> reserved for connector-specific search params
```

The real SDK adapter currently uses the Milvus SDK `IndexFlatSearchParam` and returns one result set for one query vector. Advanced Milvus search params, metadata filters, partitions, and multi-query batching are intentionally deferred until the source-side seed/search loop is stable.

The connector returns ranked hits in Milvus result order:

```go
SearchResponse{
    Hits: []SearchHit{
        {ID: "vec-000001", Rank: 1, Score: 0.98},
    },
}
```

`ExpandK` is used as the search limit so the fingerprint builder can observe boundary candidates around the business `TopK` cutoff.

## Score normalization

The project-wide normalized score rule is:

```text
larger SearchHit.Score means better match
```

Milvus metric handling:

```text
cosine: pass score through
ip:     pass score through
l2:     convert distance to negative score
```

This keeps Milvus output comparable with pgvector and future connectors.

## Safe identifiers

Milvus collection and field names are restricted to simple identifiers:

```text
^[A-Za-z_][A-Za-z0-9_]*$
```

Supported examples:

```text
items
embedding
id
```

Rejected examples:

```text
items;drop
public.items
"items"
```

Although Milvus is not SQL, rejecting unsafe dynamic names prevents accidental SDK misuse and keeps configuration behavior predictable.

## MVP role

This connector is the source-side search connector for the migration MVP. The intended later flow is:

```text
synthetic fixture records
        ↓
seed Milvus collection
        ↓
Milvus Search(query vectors)
        ↓
connectors.SearchResponse
        ↓
fingerprint artifact builder
        ↓
verification runner
```

After the real SDK adapter and fixture seeding are added, this connector will feed source-side retrieval behavior into the same artifact comparison path already used by local offline verification.
