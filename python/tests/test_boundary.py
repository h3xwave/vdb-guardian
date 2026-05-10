import pytest

from vdb_fingerprint_engine.boundary import select_boundary_candidates
from vdb_fingerprint_engine.schemas import SearchHit


def test_select_boundary_candidates_uses_rank_window_and_score_delta() -> None:
    hits = [
        SearchHit(id="doc-1", rank=1, score=0.99),
        SearchHit(id="doc-2", rank=2, score=0.95),
        SearchHit(id="doc-3", rank=3, score=0.90),
        SearchHit(id="doc-4", rank=4, score=0.885),
        SearchHit(id="doc-5", rank=5, score=0.880),
    ]

    candidates = select_boundary_candidates(hits, top_k=3, rank_before_k=1, delta=0.02)

    assert [candidate.id for candidate in candidates] == ["doc-3", "doc-4", "doc-5"]


def test_select_boundary_candidates_returns_empty_for_empty_hits() -> None:
    assert select_boundary_candidates([], top_k=3, rank_before_k=1, delta=0.02) == []


def test_select_boundary_candidates_rejects_invalid_top_k() -> None:
    with pytest.raises(ValueError, match="top_k"):
        select_boundary_candidates([], top_k=0, rank_before_k=1, delta=0.02)
