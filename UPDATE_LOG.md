# Update Log

## 2026-06-09 - Simplify PLATES Command UX

Summary:
- Simplified PLATES into a cleaner render-only command-template shell.
- Reorganized help into purpose-based sections with variables and state first.
- Added `shell clear` for clearing the terminal screen.
- Changed user-facing terminology from `ingredients` to `options`.
- Added preferred `options:` YAML schema while keeping legacy `ingredients:` loading compatibility.
- Updated default rack plates to use `options:`.

Renamed commands:
- `show ingredients` -> `show options`
- `show plate` -> `info` and `ll`
- `output history` -> `history`

Removed user-facing commands:
- `stamp`
- `lint plate`
- `lint <plate>`
- `lint rack`
- `explain lint`
- `health`
- `output show <id>`
- `output repeat <id>`
- `output delete <id>`
- `output clear`
- `output stats`
- `tip`
- `fortune`
- `random plate`
- `random plate --use`
- `version`
- `about`

Notes:
- `render`, `r`, and `run` all render only and never execute commands.
- Internal validation remains available for loading and pack workflows.
