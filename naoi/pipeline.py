"""Contains functionality related to naoi pipeline configuration files."""

from pathlib import Path
from typing import Iterator, TextIO

from blake3 import blake3
from pydantic import BaseModel, Field, ValidationError, field_validator
from yaml import YAMLError, safe_load

from naoi.exceptions import PipelineParseError, PipelineValidationError

TRIGGERS = frozenset(("push", "merge_request", "tag"))


VariableType = str | int | float | bool
"""Represents a type that a job variable can have in a naoi config file."""


class CacheKey(BaseModel):
    """Represents a complex cache key in a naoi job's cache configuration."""

    files: list[str]


class CacheConfig(BaseModel):
    """Represents the configuration for a naoi job's cache."""

    key: str | CacheKey
    paths: list[str]


class Job(BaseModel):
    """Represents the configuration of a specific job in naoi."""

    image: str
    before_script: list[str] = []
    script: list[str] = []
    after_script: list[str] = []
    variables: dict[str, VariableType] = {}
    cache: CacheConfig | None = None
    condition: str | None = Field(default=None, alias="if")

    def scripts(self) -> Iterator[str]:
        """Iterates through all scripts for this job in sequence."""
        for script in [self.before_script, self.script, self.after_script]:
            yield from script


class PipelineTrigger(BaseModel):
    """Represents the configuration of pipeline triggers in naoi."""

    event: str
    branches: list[str] = []
    condition: str | None = Field(default=None, alias="if")

    @field_validator("event")
    @classmethod
    def event_must_be_valid(cls, v: str):
        if v not in TRIGGERS:
            raise ValueError(f"Event must be one of {', '.join(TRIGGERS)}")
        return v


def _hydrate_job(
    job: dict[str, any], default: dict[str, any], variables: dict[str, VariableType]
):
    """
    Updates an individual job definition with the given defaults and default variables.
    """
    for key in [
        "image",
        "before_script",
        "script",
        "after_script",
        "cache",
        "variables",
    ]:
        if key not in job and key in default:
            job[key] = default[key]
    if "variables" not in job:
        job["variables"] = {}
    for variable in variables.keys():
        if variable not in job["variables"]:
            job["variables"][variable] = variables[variable]


def _process_inline_jobs(yaml: dict[str, any]):
    """
    Refactors the given YAML dictionary so that inline job definitions are instead
    defined under the job key, and they are hydrated to include the given defaults
    and default variables.
    """
    jobs: dict[str, any] = {}
    for key in list(yaml.keys()):
        if key not in ["name", "when", "steps", "variables", "default"]:
            jobs[key] = yaml.pop(key)
            if not isinstance(jobs[key], dict):
                raise PipelineValidationError(
                    f"Job configuration must be a dictionary at jobs.[{key}]."
                )
            if "default" in yaml:
                _hydrate_job(
                    jobs[key], yaml.get("default", {}), yaml.get("variables", {})
                )
    yaml.pop("default", None)
    yaml.pop("variables", None)
    yaml["jobs"] = jobs


def _yaml_location_to_string(loc: tuple[str | int, ...], link: str = ".") -> str:
    """Returns a pydantic location as a human-readable location string."""
    return link.join([f"[{l}]" if isinstance(l, int) else l for l in loc])


def _parse_validation_error(ve: ValidationError) -> PipelineValidationError:
    """
    Parses a pydantic ValidationError and returns a human-readable
    NaoiConfigValidationError.
    """
    causes = [
        f"\n   • {ed['msg']} at {_yaml_location_to_string(ed['loc'])}"
        for ed in ve.errors()
    ]
    return PipelineValidationError(
        f"Pipeline configuration was invalid: {''.join(causes)}."
    )


class Pipeline(BaseModel):
    """
    Represents the configuration of a naoi pipeline, as expressed in a naoi YAML file.
    """

    identifier: str = Field(init=False, default="")

    name: str
    when: list[PipelineTrigger]
    steps: list[str]
    jobs: dict[str, Job]

    trigger: str = Field(init=False, default="manual")
    trigger_context: dict[str, VariableType] = Field(init=False, default={})

    @classmethod
    def from_yaml(
        cls, stream: bytes | str | TextIO, identifier: str | None = None
    ) -> "Pipeline":
        """
        Returns a Pipeline object containing the configuration specified by the given
        YAML.

        :param stream: The contents of the YAML configuration file to parse.
        :param identifier: Optionally, the identifier to give the generated pipeline.
        :return: The Pipeline object representing the given pipeline configuration.
        """
        try:
            yaml = safe_load(stream)
        except YAMLError:
            raise PipelineParseError(
                "Pipeline configuration could not be parsed as valid YAML."
            )
        if not isinstance(yaml, dict):
            raise PipelineParseError(
                "Pipeline configuration must be in form of dictionary."
            )
        _process_inline_jobs(yaml)
        try:
            pipeline = Pipeline(**yaml)
            pipeline.set_identifier(identifier)
            return pipeline
        except ValidationError as ve:
            raise _parse_validation_error(ve)

    @classmethod
    def from_file(cls, filename: Path, identifier: str | None = None) -> "Pipeline":
        """
        Returns a Pipeline object containing the configuration specified in the given
        YAML file.

        :param filename: A Path representing the filename of the YAML configuration file
            to parse.
        :param identifier: Optionally, the identifier to give the generated pipeline.
        :return: The Pipeline object representing the given pipeline file.
        """
        try:
            with open(filename, "r") as file:
                return Pipeline.from_yaml(
                    file, identifier=identifier or str(filename.absolute())
                )
        except FileNotFoundError:
            raise PipelineParseError("Pipeline configuration file does not exist.")

    def set_identifier(self, identifier: str | None = None):
        """
        Sets the identifier for the pipeline to a blake3 hash of either the given
        identifier or an automatically generated dump of the pipeline configuration.

        :param identifier: Optionally, the identifier to use to generate the
            pipeline's hashed identifier.
        """
        if not identifier:
            identifier = self.model_dump_json()
        pipeline_hash = blake3(str.encode(identifier))
        self.identifier = pipeline_hash.hexdigest()
