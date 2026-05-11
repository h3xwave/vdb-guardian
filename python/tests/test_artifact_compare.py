import json
from pathlib import Path

import pytest

from vdb_fingerprint_engine.artifact_compare import compare_fingerprint_artifacts


def write_artifact(path: Path, fingerprints: list[dict[str, object]]) -> Path:
    path.write_text(json.dumps({"fingerprints": fingerprints}), encoding="utf-8")
    return path


def test_compare_fingerprint_artifacts_returns_perfect_score_for_identical_artifacts(
    tmp_path: Path,
) -> None:
    source = write_artifact(
        tmp_path / "source.json",
        [
            {
                "query_id": "q-1",
                "stable_neighbors": ["a", "b", "c"],
                "boundary_candidates": ["d", "e"],
                "top_k_ids": ["a", "b", "c", "d"],
            }
        ],
    )
    target = write_artifact(
        tmp_path / "target.json",
        [
            {
                "query_id": "q-1",
                "stable_neighbors": ["a", "b", "c"],
                "boundary_candidates": ["d", "e"],
                "top_k_ids": ["a", "b", "c", "d"],
            }
        ],
    )

    output = compare_fingerprint_artifacts("job-1", source, target)

    assert output.job_id == "job-1"
    assert output.consistency_score == 1.0
    assert output.metrics.fingerprint_distance == 0.0
    assert output.metrics.stable_neighbor_distance == 0.0
    assert output.metrics.boundary_candidate_distance == 0.0
    assert output.metrics.boundary_flip_rate == 0.0
    assert output.metrics.matched_query_count == 1
    assert output.metrics.missing_source_query_count == 0
    assert output.metrics.missing_target_query_count == 0


def test_compare_fingerprint_artifacts_detects_stable_neighbor_and_boundary_changes(
    tmp_path: Path,
) -> None:
    source = write_artifact(
        tmp_path / "source.json",
        [
            {
                "query_id": "q-1",
                "stable_neighbors": ["a", "b", "c"],
                "boundary_candidates": ["d", "e"],
                "top_k_ids": ["a", "b", "c", "d"],
            }
        ],
    )
    target = write_artifact(
        tmp_path / "target.json",
        [
            {
                "query_id": "q-1",
                "stable_neighbors": ["a", "b", "x"],
                "boundary_candidates": ["d", "f"],
                "top_k_ids": ["a", "b", "x", "f"],
            }
        ],
    )

    output = compare_fingerprint_artifacts("job-1", source, target)

    assert output.consistency_score < 1.0
    assert output.metrics.fingerprint_distance > 0.0
    assert output.metrics.stable_neighbor_distance > 0.0
    assert output.metrics.boundary_candidate_distance > 0.0
    assert output.metrics.boundary_flip_rate > 0.0
    assert output.metrics.matched_query_count == 1


def test_compare_fingerprint_artifacts_reports_missing_queries(tmp_path: Path) -> None:
    source = write_artifact(
        tmp_path / "source.json",
        [
            {
                "query_id": "q-1",
                "stable_neighbors": ["a"],
                "boundary_candidates": ["b"],
                "top_k_ids": ["a", "b"],
            }
        ],
    )
    target = write_artifact(
        tmp_path / "target.json",
        [
            {
                "query_id": "q-2",
                "stable_neighbors": ["z"],
                "boundary_candidates": ["y"],
                "top_k_ids": ["z", "y"],
            }
        ],
    )

    output = compare_fingerprint_artifacts("job-1", source, target)

    assert output.metrics.matched_query_count == 0
    assert output.metrics.missing_source_query_count == 1
    assert output.metrics.missing_target_query_count == 1
    assert output.metrics.fingerprint_distance == 1.0
    assert output.consistency_score == 0.0


def test_compare_fingerprint_artifacts_rejects_empty_artifacts(tmp_path: Path) -> None:
    source = write_artifact(tmp_path / "source.json", [])
    target = write_artifact(tmp_path / "target.json", [])

    with pytest.raises(ValueError, match="fingerprints"):
        compare_fingerprint_artifacts("job-1", source, target)
