# Fingerprint Engine Specification

The fingerprint engine computes retrieval behavior differences between source and target vector databases.

## Initial metrics

The first implementation includes:

- Boundary candidate selection.
- Jaccard distance.
- Boundary flip rate.
- Weighted fingerprint distance.

## Boundary candidates

Boundary candidates are hits near the topK decision boundary whose score is close to the K-th result. They are important because migration-related indexing, distance, or filtering differences often cause these candidates to enter or leave visible topK results.

## Distance metrics

The engine returns normalized values in `[0, 1]` where possible. Lower fingerprint distance means source and target retrieval behavior are more similar. Higher consistency score means better migration consistency.

## Protocol direction

The Go control plane will provide artifact paths or JSON payloads. The Python engine will return a compact JSON summary and write detailed artifacts through the artifact boundary.
