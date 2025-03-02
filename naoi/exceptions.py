"""Contains classes to encode specific exceptions that can be raised by naoi."""

class CacheError(RuntimeError):
    """Raised when the cache encounters a runtime error."""

    pass

class PipelineValidationError(ValueError):
    """Raised when a validation error occurs during parsing of a naoi config file."""

    pass


class PipelineParseError(ValueError):
    """Raised when a parse error occurs during parsing of a naoi config file."""

    pass


class RunnerError(RuntimeError):
    """Raised when a naoi runner encounters a fatal runtime exception."""

    pass
