# omisego-doc

Documentation for the OmiseGO Project

https://omisego.immutability.io/

## Build

Documentation is written using Sphinx.  To build:

```
pip install sphinx sphinx_rtd_theme recommonmark
cd docs
sphinx-build -d _build/doctrees . _build/html
```

Docs will be in `docs/build/html`.  To view the documentation open `docs/build/html/index.html` with a browser.