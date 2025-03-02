RUN = poetry run
PYTHON = $(RUN) python

DOCS_SRC = docs
DOCS_OUT = docs/_build

.PHONY: docs serve-docs

docs:
	@echo "📜 Building documentation..."
	@$(RUN) sphinx-build -M html $(DOCS_SRC) $(DOCS_OUT)

serve-docs:
	@echo "📜 Building and serving documentation..."
	@$(RUN) sphinx-autobuild $(DOCS_SRC) $(DOCS_OUT) --watch naoi
