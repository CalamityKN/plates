# Safety

PLATES renders commands only.

PLATES does not execute:

- rendered command output
- plate templates
- Forge Mode draft lines
- saved YAML files

Always review rendered output before copying it into a terminal.

## Secrets

Pantry and workspace YAML files are plain text. Store secrets carefully and avoid saving passwords in workspace files when possible.

For sensitive values, use `secret set <key> <value>`. The Phase 10 secret store lives in `data/secrets/secrets.yaml` and is local plaintext unless future encryption is added. The directory is ignored by git by default.

Renders containing secrets are stored redacted in output history by default. `store_secret_outputs: true` allows raw secret output to be stored, and should be used carefully.

## Responsibility

PLATES is a local rendering and organization tool. The user remains responsible for deciding whether a rendered command is appropriate and safe to run.
