# NightOps Documentation Site

The public documentation is built with MkDocs Material from `docs/site/` and
published to GitHub Pages by `.github/workflows/docs.yml`.

The published navigation covers:

- mission-control overview and installation
- first mission workflow
- Obsidian interoperability
- architecture and development boundaries
- SkyBase Atlas contribution workflow
- offline/sync behavior and FAQ
- contributor expectations and roadmap

Run a local preview when MkDocs Material is installed:

```sh
python3 -m pip install --requirement requirements-docs.txt
mkdocs serve
```

The generated `site/` directory is intentionally not committed.
