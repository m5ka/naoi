"""
Contains functionality for using naoi as a command-line tool.

This is best run as a Python script, i.e. ``poetry run naoi`` or ``python -m naoi.cli``,
and so should not be called as code.
"""

import sys
from pathlib import Path

import click
from rich.pretty import pprint

from naoi.pipeline import Pipeline
from naoi.runner import Runner
from naoi.exceptions import PipelineParseError, PipelineValidationError


@click.group
def naoi():
    """Entry-point for naoi's command-line application."""
    pass


@naoi.command(help="Runs the given naoi config file as a pipeline")
@click.argument("filename", type=click.Path(exists=True, path_type=Path))
def run(filename: Path):
    try:
        pipeline = Pipeline.from_file(filename)
    except (PipelineParseError, PipelineValidationError) as e:
        print(f"❌ {str(e)}", file=sys.stderr)
        sys.exit(1)
    runner = Runner.from_pipeline(pipeline)
    for result in runner.execute():
        pprint(result)


@naoi.command(help="Parse the given naoi config file to ensure it's valid")
@click.argument("filename", type=click.Path(exists=True, path_type=Path))
@click.option(
    "-v", "--verbose", is_flag=True, help="Pretty-print the config after validating"
)
def parse(filename: Path, verbose: bool):
    try:
        config = Pipeline.from_file(filename)
    except (PipelineParseError, PipelineValidationError) as e:
        print(f"❌ {str(e)}", file=sys.stderr)
        sys.exit(1)
    if verbose:
        pprint(config)


if __name__ == "__main__":
    naoi()
