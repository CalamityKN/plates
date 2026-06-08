# Variables

PLATES resolves template variables from pantry values, workspace values, session updates, and ingredient defaults.

## Precedence

Highest priority wins:

1. Session/workspace variables set during the active workspace session
2. Workspace variables from `data/workspaces/<workspace>.yaml`
3. Pantry/global variables from `data/pantry/globals.yaml`
4. Ingredient defaults from the plate

## Pantry

Use pantry variables for values shared across workspaces.

```text
setg my_ip 10.10.14.3
setg http_port 8000
show pantry
```

## Workspace

Use workspace variables for target-specific values.

```text
workspace devhub
set target 10.129.202.242
set workdir C:\Users\knjoh\code\boxes\devhub
show workspace
```

## Ingredients

`show ingredients` displays required and optional ingredients for the loaded plate. Missing required ingredients prevent `stamp` and `render` from printing rendered output.

Defaults are used when no session, workspace, or pantry value exists.

## Secrets

Use secrets for sensitive values:

```text
secret set password SuperSecret123
secret list
secret reveal password
```

Template syntax:

```text
{{secret "password"}}
```

Secret ingredients can be marked with `secret: true`. PLATES masks those values in normal displays.

Secrets are stored in `data/secrets/secrets.yaml` as local plaintext in this MVP.

## Environment Variables

Templates can read environment variables:

```text
{{env "HOME"}}
{{env "PLATES_TOKEN"}}
```

Missing environment variables cause rendering to fail with a useful error.
