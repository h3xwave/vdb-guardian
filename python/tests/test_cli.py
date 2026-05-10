import json
import subprocess
import sys
from pathlib import Path


def test_cli_version_outputs_engine_name() -> None:
    result = subprocess.run(
        [sys.executable, "-m", "vdb_fingerprint_engine.cli", "--version"],
        check=False,
        capture_output=True,
        text=True,
    )

    assert result.returncode == 0
    assert "vdb-fingerprint-engine" in result.stdout


def test_cli_compare_writes_output_json(tmp_path: Path) -> None:
    input_path = tmp_path / "input.json"
    output_path = tmp_path / "output.json"
    input_path.write_text(
        json.dumps(
            {
                "job_id": "job-1",
                "source_fingerprint_path": "source.json",
                "target_fingerprint_path": "target.json",
            }
        ),
        encoding="utf-8",
    )

    result = subprocess.run(
        [
            sys.executable,
            "-m",
            "vdb_fingerprint_engine.cli",
            "compare",
            "--input",
            str(input_path),
            "--output",
            str(output_path),
        ],
        check=False,
        capture_output=True,
        text=True,
    )

    assert result.returncode == 0, result.stderr
    output = json.loads(output_path.read_text(encoding="utf-8"))
    assert output["job_id"] == "job-1"
    assert output["consistency_score"] == 1.0
    assert output["metrics"]["fingerprint_distance"] == 0.0
    assert output["metrics"]["boundary_flip_rate"] == 0.0


def test_cli_compare_rejects_missing_input(tmp_path: Path) -> None:
    output_path = tmp_path / "output.json"

    result = subprocess.run(
        [
            sys.executable,
            "-m",
            "vdb_fingerprint_engine.cli",
            "compare",
            "--input",
            str(tmp_path / "missing.json"),
            "--output",
            str(output_path),
        ],
        check=False,
        capture_output=True,
        text=True,
    )

    assert result.returncode != 0
    assert "input" in result.stderr.lower()
    assert not output_path.exists()
