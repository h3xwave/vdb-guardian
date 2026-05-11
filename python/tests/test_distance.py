import pytest

from vdb_fingerprint_engine.distance import (
    boundary_flip_rate,
    jaccard_distance,
    weighted_fingerprint_distance,
)


def test_jaccard_distance_returns_zero_for_two_empty_sets() -> None:
    assert jaccard_distance(set(), set()) == 0.0


def test_jaccard_distance_returns_zero_for_identical_sets() -> None:
    assert jaccard_distance({"a", "b"}, {"a", "b"}) == 0.0


def test_jaccard_distance_returns_one_for_disjoint_sets() -> None:
    assert jaccard_distance({"a"}, {"b"}) == 1.0


def test_boundary_flip_rate_counts_boundary_items_crossing_top_k() -> None:
    source_top_k = {"a", "b"}
    target_top_k = {"b", "c"}
    boundary_candidates = {"a", "b", "c", "d"}

    assert boundary_flip_rate(source_top_k, target_top_k, boundary_candidates) == 0.5


def test_weighted_fingerprint_distance_combines_named_components() -> None:
    distance = weighted_fingerprint_distance(
        components={"stable": 0.1, "boundary": 0.5},
        weights={"stable": 0.25, "boundary": 0.75},
    )

    assert distance == pytest.approx(0.4)


def test_weighted_fingerprint_distance_rejects_missing_weights() -> None:
    with pytest.raises(ValueError, match="weights"):
        weighted_fingerprint_distance(components={"stable": 0.1}, weights={})
