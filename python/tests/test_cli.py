import subprocess
import sys


def test_cli_version_outputs_engine_name() -> None:
    result = subprocess.run(
        [sys.executable, "-m", "vdb_fingerprint_engine.cli", "--version"],
        check=False,
        capture_output=True,
        text=True,
    )

    assert result.returncode == 0
    assert "vdb-fingerprint-engine" in result.stdout
