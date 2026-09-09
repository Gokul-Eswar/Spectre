import pytest
from analyzer.vector_store import search_evidence, index_evidence


def test_vector_store_when_chromadb_missing(monkeypatch):
    import analyzer.vector_store as vs

    monkeypatch.setattr(vs, "get_client", lambda case_id: None)

    res = search_evidence("test_case", "query")
    assert res["status"] == "error"
    assert "chromadb is not installed" in res["error"]

    idx_res = index_evidence("test_case", ["foo.json"])
    assert idx_res["status"] == "error"
    assert "chromadb is not installed" in idx_res["error"]


def test_search_evidence_empty():
    pytest.importorskip("chromadb")
    case_id = "non_existent_case_999"
    result = search_evidence(case_id, "search query")
    assert result["status"] == "success"
    assert result["results"] == []


def test_index_evidence_nonexistent_files():
    pytest.importorskip("chromadb")
    case_id = "test_case_empty_files"
    result = index_evidence(case_id, ["/non_existent_path/foo.json"])
    assert result["status"] == "success"
    assert result["indexed_count"] == 0
