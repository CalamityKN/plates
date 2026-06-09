# Phase 1

Phase 1 establishes the PLATES shell and storage foundation:

- Interactive shell with history, help, exit, and quit.
- Idempotent `init` command for `data/pantry`, `data/workspaces`, and `data/rack`.
- YAML-backed global pantry variables.
- YAML-backed workspace variables.
- Recursive plate discovery under `data/rack`.

Rendering, validation of plate options, forge mode, search, linting, clipboard support, and startup banners are reserved for later phases.
