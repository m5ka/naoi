from naoi import __version__ as naoi_version


extensions = [
    "myst_parser",
    "sphinx.ext.autodoc",
    "sphinx.ext.autosummary",
    "sphinx.ext.doctest",
    "sphinx.ext.todo",
]

project = "naoi"
version = naoi_version

html_theme = "sphinx_rtd_theme"

autodoc_typehints = "both"
