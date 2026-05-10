"""Distance metrics for retrieval behavior fingerprint comparison."""


def jaccard_distance(left: set[str], right: set[str]) -> float:
    """Compute normalized Jaccard distance between two identifier sets.

    This metric is used to quantify differences between stable-neighbor sets or
    boundary-candidate sets. A value of 0.0 means the sets are equivalent, while
    a value of 1.0 means they have no overlap.

    Args:
        left: First set of vector identifiers.
        right: Second set of vector identifiers.

    Returns:
        A normalized distance in the inclusive range [0.0, 1.0].
    """
    if not left and not right:
        return 0.0

    union = left | right
    intersection = left & right
    return 1.0 - (len(intersection) / len(union))


def boundary_flip_rate(
    source_top_k: set[str], target_top_k: set[str], boundary_candidates: set[str]
) -> float:
    """Measure how often boundary candidates cross the topK visibility boundary.

    A boundary flip occurs when a candidate is visible in the source topK but not
    the target topK, or visible in the target topK but not the source topK. This
    metric is central to vdb-guardian because it detects migration drift that a
    coarse data-count check or average benchmark may miss.

    Args:
        source_top_k: Identifiers visible in the source database topK result.
        target_top_k: Identifiers visible in the target database topK result.
        boundary_candidates: Candidate identifiers near the source or target topK cutoff.

    Returns:
        The fraction of boundary candidates whose topK visibility changed.
    """
    if not boundary_candidates:
        return 0.0

    flipped = [
        candidate_id
        for candidate_id in boundary_candidates
        if (candidate_id in source_top_k) != (candidate_id in target_top_k)
    ]
    return len(flipped) / len(boundary_candidates)


def weighted_fingerprint_distance(
    *, components: dict[str, float], weights: dict[str, float]
) -> float:
    """Combine named fingerprint-distance components with normalized weights.

    Args:
        components: Mapping from component names to normalized distances.
        weights: Mapping from component names to non-negative weights.

    Returns:
        Weighted normalized fingerprint distance.

    Raises:
        ValueError: If components are missing weights, weights are negative, or
            the total weight is not positive.
    """
    if not components:
        return 0.0

    missing_weights = set(components) - set(weights)
    if missing_weights:
        raise ValueError(f"weights missing components: {sorted(missing_weights)}")

    total_weight = 0.0
    total_distance = 0.0
    for name, component_distance in components.items():
        weight = weights[name]
        if weight < 0:
            raise ValueError(f"weights must not be negative: {name}")
        total_weight += weight
        total_distance += component_distance * weight

    if total_weight <= 0:
        raise ValueError("weights must have a positive total")

    return total_distance / total_weight
