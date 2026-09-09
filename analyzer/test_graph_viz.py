import os
import tempfile
from analyzer.graph_viz import generate_visual_report


def test_generate_visual_report():
    data = {
        "case_name": "Test Case",
        "case_id": "test_case_123",
        "entities": [
            {"id": "e1", "type": "domain", "value": "example.com", "source": "dns"},
            {"id": "e2", "type": "ip", "value": "93.184.216.34", "source": "dns"},
        ],
        "relationships": [
            {"from_entity_id": "e1", "to_entity_id": "e2", "type": "resolves_to"}
        ],
    }

    with tempfile.TemporaryDirectory() as tmpdir:
        target_path = os.path.join(tmpdir, "test_report.html")
        data["output_path"] = target_path

        result = generate_visual_report(data)
        assert result["status"] == "success"
        assert os.path.exists(target_path)
        with open(target_path, "r", encoding="utf-8") as f:
            content = f.read()
            assert "example.com" in content
