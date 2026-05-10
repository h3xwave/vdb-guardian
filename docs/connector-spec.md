# Connector Specification

Connectors normalize vector database behavior for the Go control plane and Python fingerprint engine.

## Required behavior

A connector must:

1. Connect to a database with context cancellation.
2. Count records in a collection or table.
3. Execute normalized vector search requests.
4. Return ranked hits with stable identifiers, ranks, scores, and optional metadata.
5. Close resources safely.

## Normalization rule

Database-specific SDK objects must not leak into core packages. Milvus, pgvector, and future connectors must convert native results into the shared `SearchResponse` shape before handing results to the rest of the system.

## Future connectors

Planned connectors include:

- Milvus
- pgvector
- Qdrant
- Weaviate
- Elastic/OpenSearch
- Pinecone
