"""Shared schemas for retrieval behavior fingerprint calculations.

The Python engine keeps these schemas small and explicit so the Go control plane
can exchange JSON payloads with the algorithm layer without depending on Python
implementation details.
"""

from pydantic import BaseModel, Field


class SearchHit(BaseModel):
    """Represent one normalized vector search result.

    Args:
        id: Stable vector or document identifier used to compare source and target results.
        rank: One-based rank returned by a vector database search.
        score: Similarity score used to identify boundary candidates and score curves.
    """

    id: str
    rank: int = Field(ge=1)
    score: float


class CompareInput(BaseModel):
    """Represent the JSON request sent from the Go control plane to Python.

    Args:
        job_id: Stable verification job identifier used to correlate input and output.
        source_fingerprint_path: Artifact path for the source retrieval behavior fingerprint.
        target_fingerprint_path: Artifact path for the target retrieval behavior fingerprint.
    """

    job_id: str
    source_fingerprint_path: str
    target_fingerprint_path: str


class MetricSummary(BaseModel):
    """Represent normalized comparison metrics returned to the Go control plane.

    Args:
        fingerprint_distance: Overall normalized distance between source and target fingerprints.
        boundary_flip_rate: Fraction of boundary candidates whose topK visibility changed.
    """

    fingerprint_distance: float = Field(ge=0.0, le=1.0)
    boundary_flip_rate: float = Field(ge=0.0, le=1.0)


class CompareOutput(BaseModel):
    """Represent the JSON response produced by the Python fingerprint engine.

    Args:
        job_id: Verification job identifier copied from the compare input.
        consistency_score: Normalized consistency score where higher means more consistent.
        metrics: Decomposed fingerprint comparison metrics.
    """

    job_id: str
    consistency_score: float = Field(ge=0.0, le=1.0)
    metrics: MetricSummary
