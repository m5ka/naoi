"""
Contains functionality for the execution of naoi pipeline scripts by specific individual
units known as runners.
"""

from collections.abc import Callable
from dataclasses import dataclass, field
from typing import Iterator, Type

from docker import DockerClient
from docker.errors import ImageNotFound, APIError
from docker.models.containers import Container

from naoi import settings
from naoi.exceptions import CacheError, RunnerError
from naoi.pipeline import CacheKey, Pipeline, Job


@dataclass(frozen=True)
class ScriptResult:
    exit_code: int
    output: str


@dataclass(frozen=True)
class JobResult:
    success: bool
    script_results: list[ScriptResult] = field(default_factory=list)
    error_message: str | None = None


def _require_container(
    error_class: Type[Exception] = RunnerError, error_message: str | None = None
) -> Callable:
    def __require_container(func: Callable) -> Callable:
        def ___require_container(self, *args, **kwargs):
            if self._container is None:
                raise error_class(error_message or "No container is active in Runner.")
            return func(self, *args, **kwargs)

        return ___require_container

    return __require_container


def _require_job(
    error_class: Type[Exception] = RunnerError, error_message: str | None = None
) -> Callable:
    def __require_job(func: Callable) -> Callable:
        def ___require_job(self, *args, **kwargs):
            if self._job is None:
                raise error_class(error_message or "No job is active in Runner.")
            return func(self, *args, **kwargs)

        return ___require_job

    return __require_job


class Runner:
    def __init__(self, client: DockerClient, pipeline: Pipeline):
        self.client = client
        self.pipeline = pipeline

        self._container: Container | None = None
        self._job: Job | None = None

    @classmethod
    def from_pipeline(cls, pipeline: Pipeline) -> "Runner":
        return Runner(
            client=DockerClient(base_url=settings.docker_host), pipeline=pipeline
        )

    @_require_job(
        error_message="Cannot start a container without specifying an active job."
    )
    def _start_container(self):
        if self._container is not None:
            raise RunnerError(
                "Cannot start a container while another is active in a Runner"
            )
        self._container = self.client.containers.run(
            self._job.image,
            detach=True,
            tty=True,
            environment={
                "TRIGGER": self.pipeline.trigger,
                **self.pipeline.trigger_context,
                **self._job.variables,
            },
        )

    @_require_container(
        error_message="Cannot configure container: no container is active in Runner."
    )
    @_require_job(
        error_message="Cannot configure container: no job is active in Runner."
    )
    def _setup_container(self):
        # todo: setup storage/pull in files
        self._configure_cache()

    @_require_container(
        error_class=CacheError,
        error_message="Cannot configure cache: no container is active in Runner.",
    )
    @_require_job(
        error_class=CacheError,
        error_message="Cannot configure cache: no job is active in Runner.",
    )
    def _configure_cache(self):
        if self._job.cache is None:
            raise CacheError(
                "Cannot configure cache: cache is not specified in job configuration."
            )
        key = self._job.cache.key
        if isinstance(key, CacheKey):
            for file in key.files:
                exit_code, output = self._container.exec_run(
                    ["sha256sum", file, "|", "awk", "{print $1}"]
                )
                if int(exit_code) != 0:
                    raise CacheError(f"Cache failed: could not hash file {file}.")
                print(output)

    @_require_container(
        error_message="Cannot stop container: no container is active in Runner."
    )
    def _stop_container(self, remove=True):
        self._container.stop()
        if remove:
            self._container.remove()
            self._container = None

    @_require_container(
        error_message="Cannot execute commands: no container is active in Runner."
    )
    @_require_job(error_message="Cannot execute commands: no job is active in Runner.")
    def _execute_commands(self) -> JobResult:
        script_results: list[ScriptResult] = []
        success = True
        error_message: str | None = None
        for script in self._job.scripts():
            exit_code, raw_output = self._container.exec_run(
                ["/bin/bash", "-c", script]
            )
            script_results.append(ScriptResult(int(exit_code), raw_output.decode()))
            if exit_code != 0:
                success = False
                error_message = "Script returned a non-zero exit code"
                break
        return JobResult(
            success=success, script_results=script_results, error_message=error_message
        )

    def execute(self) -> Iterator[JobResult]:
        for step in self.pipeline.steps:
            self._job = self.pipeline.jobs[step]
            try:
                self._start_container()
            except APIError:
                yield JobResult(
                    success=False, error_message="Invalid container specification"
                )
                return
            except ImageNotFound:
                yield JobResult(
                    success=False,
                    error_message=f"No such image found: {self._job.image}",
                )
                return
            try:
                result = self._execute_commands()
                yield result
                if not result.success:
                    return
            finally:
                self._stop_container()
