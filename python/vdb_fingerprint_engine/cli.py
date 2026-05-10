"""Command-line entrypoint for the Python fingerprint engine."""

import argparse

from vdb_fingerprint_engine import __version__


def build_parser() -> argparse.ArgumentParser:
    """Build the command-line parser for the fingerprint engine.

    Returns:
        An argparse parser that currently supports version output and can later
        be extended with JSON-based compare commands invoked by the Go control
        plane.
    """
    parser = argparse.ArgumentParser(prog="vdb-fingerprint-engine")
    parser.add_argument("--version", action="store_true", help="print engine version and exit")
    return parser


def main() -> int:
    """Run the fingerprint engine command line interface.

    Returns:
        Process exit code. A zero value indicates successful command handling.
    """
    parser = build_parser()
    args = parser.parse_args()
    if args.version:
        print(f"vdb-fingerprint-engine {__version__}")
        return 0

    parser.print_help()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
