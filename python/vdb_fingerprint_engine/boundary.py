"""Boundary candidate selection for retrieval behavior fingerprints."""

from vdb_fingerprint_engine.schemas import SearchHit


def select_boundary_candidates(
    hits: list[SearchHit], *, top_k: int, rank_before_k: int, delta: float
) -> list[SearchHit]:
    """Select candidates near the topK decision boundary.

    The boundary candidate set captures vectors whose ranks are close to the
    visible topK cutoff and whose score is close to the K-th result. These items
    are especially useful for detecting migration drift because they can enter or
    leave topK when source and target vector databases differ in indexing,
    distance calculation, or filtering behavior.

    Args:
        hits: Ranked search hits from a topK expanded query result.
        top_k: Business-visible topK cutoff. Must be greater than zero.
        rank_before_k: Number of ranks before K included in the observation window.
        delta: Maximum absolute score difference from the K-th result.

    Returns:
        Search hits that fall inside the rank window and score-delta threshold.

    Raises:
        ValueError: If top_k is not positive, rank_before_k is negative, or delta is negative.
    """
    if top_k <= 0:
        raise ValueError("top_k must be greater than zero")
    if rank_before_k < 0:
        raise ValueError("rank_before_k must not be negative")
    if delta < 0:
        raise ValueError("delta must not be negative")
    if not hits:
        return []

    sorted_hits = sorted(hits, key=lambda hit: hit.rank)
    kth_hit = next((hit for hit in sorted_hits if hit.rank == top_k), None)
    if kth_hit is None:
        raise ValueError("top_k must reference an existing hit rank")

    min_rank = max(1, top_k - rank_before_k)
    tolerance = 1e-12
    return [
        hit
        for hit in sorted_hits
        if hit.rank >= min_rank and abs(hit.score - kth_hit.score) <= delta + tolerance
    ]
