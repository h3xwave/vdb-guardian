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
