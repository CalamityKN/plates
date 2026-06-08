# PLATES

PLATES is a local command-template rendering framework. It loads reusable command templates, resolves variables, and prints rendered command text for the user to copy manually.

PLATES does not execute commands.

## Build

```powershell
go mod tidy
go test ./...
go build ./...
go run .
```

## Directory Structure

```text
plates/
|-- cmd/                  Cobra CLI entrypoint
|-- internal/
|   |-- shell/            Interactive shell command parsing and presentation
|   |-- config/           Paths and YAML variable persistence
|   |-- plates/           Plate models, rack browsing, loading, resolution, rendering
|   `-- workspace/        Active workspace session state
|-- data/
|   |-- pantry/           Global variables, stored in globals.yaml
|   |-- workspaces/       Workspace YAML files
|   `-- rack/             Plate YAML files discovered recursively
|-- docs/
|-- main.go
|-- go.mod
`-- README.md
```

## Plate YAML Format

Plate files live under `data/rack/` and use `.yml` or `.yaml`.

```yaml
name: nmap_full_tcp
category: scanning
description: Full TCP port scan with output files
tags:
  - nmap
  - scanning
  - tcp
ingredients:
  target:
    description: Target IP address or hostname
    required: true
  workdir:
    description: Working directory for output files
    required: true
  rate:
    description: Minimum packet rate
    required: false
    default: "5000"
template: |
  sudo nmap -p- --min-rate {{rate}} -oA {{workdir}}/nmap/full_tcp {{target}}
```

Required plate fields are `name`, `category`, `description`, and `template`. Ingredients are variables used by the template. Required ingredients must resolve before stamping; optional ingredients may use defaults.

Templates use Go `text/template`, with simple variable-style calls such as `{{target}}`, `{{workdir}}`, and `{{rate}}`.

## Variable Precedence

Values are resolved from lowest to highest priority:

1. Ingredient defaults from the plate YAML
2. Pantry variables from `data/pantry/globals.yaml`
3. Workspace variables from `data/workspaces/<workspace>.yaml`
4. Current session values set with `set <key> <value>`

The `set` command still writes to the active workspace YAML, while also acting as a current-session override.

## Commands

```text
init                  Create data/pantry, data/workspaces, and data/rack
workspace <name>      Switch to or create a workspace
set <key> <value>     Set a variable in the active workspace
setg <key> <value>    Set a global pantry variable
list plates           List available plates by category
search plates <query> Search plates by name, description, tags, and ingredients
use <plate>           Load a plate by name or category/name
forge                 Create a new plate
config show           Show PLATES config
config set <key> <value> Set a config value
tip                   Show a usage tip
fortune               Show a PLATES fortune
random plate          Show a random plate
random plate --use    Load a random plate
version               Show PLATES version
about                 Show PLATES summary
pack list             List exported and imported packs
pack export <name>    Export the rack as a pack
pack export <name> --category <category> Export a category
pack export <name> --tag <tag> Export plates by tag
pack export <name> --plate <category/name> Export one plate
pack inspect <path>   Inspect a pack without importing
pack validate <path>  Validate a pack
pack import <path>    Import a pack
pack import <path> --force Import and overwrite conflicts
show rack             Show rack summary
show tags             Show tag counts
show category <name>  Show plates in a category
lint plate            Lint the loaded plate
lint <plate>          Lint a plate without loading it
lint rack             Lint all plates in the rack
health                Show rack health summary
explain lint          Explain lint rules
copy                  Copy latest rendered output to clipboard
save output [file]    Save latest rendered output
output history        Show render history
output show <id>      Show a stored render
output repeat <id>    Re-display stored output
output delete <id>    Delete a stored render
output clear          Clear render history
output stats          Show output statistics
export markdown       Export latest render as Markdown
export json           Export latest render as JSON
export yaml           Export latest render as YAML
show plate            Show loaded plate metadata
show ingredients      Show variables required by current plate
stamp                 Render current plate
render                Alias for stamp
clear plate           Unload current plate
help                  Show help
exit, quit            Leave PLATES
```

## Documentation

Built-in terminal guides:

```text
guide
guide plates
guide forge
guide variables
guide rack
guide examples
guide safety
```

Reference docs:

- [docs/template_howto.md](docs/template_howto.md)
- [docs/forge_mode.md](docs/forge_mode.md)
- [docs/variables.md](docs/variables.md)
- [docs/rack_organization.md](docs/rack_organization.md)
- [docs/examples.md](docs/examples.md)
- [docs/safety.md](docs/safety.md)

Manual page:

- [man/plates.1](man/plates.1)

## Forge Mode

Forge Mode creates YAML plate templates from inside the PLATES shell. It never executes draft lines or rendered commands.

```text
PLATES > forge
FORGE[new] > set name my_plate
FORGE[my_plate] > set category scanning
FORGE[my_plate] > set description "My reusable command"
FORGE[my_plate] > add_line "tool --target {{target}}"
FORGE[my_plate] > add_var target "Target IP address"
FORGE[my_plate] > add_optional_var rate 5000 "Minimum packet rate"
FORGE[my_plate] > add_tag scanning
FORGE[my_plate] > show draft
FORGE[my_plate] > validate
FORGE[my_plate] > save
PLATES > use scanning/my_plate
```

Required ingredients are added with `add_var <name> <description>`. Optional ingredients are added with `add_optional_var <name> <default> <description>`. Variable names must be valid simple template identifiers, so `{{target}}` maps to an ingredient named `target`.

Template lines are managed with `add_line`, `insert_line`, `delete_line`, and `clear_lines`. Use `show lines`, `show vars`, `show tags`, and `show draft` to inspect the draft before saving.

Saving writes to `data/rack/<category>/<name>.yml`. Existing files are not overwritten unless you run `save --force`. After saving, Forge Mode exits and PLATES prints the `use <category>/<name>` command for the new plate.

## Linting And Validation

PLATES can audit individual plates and the whole rack without modifying files.

```text
use scanning/nmap_full_tcp
lint plate
lint scanning/nmap_full_tcp
lint rack
health
explain lint
```

`lint plate` checks the currently loaded plate. `lint <plate>` checks a specific plate by name or `category/name`. `lint rack` checks every discovered plate and prints PASS/WARN/FAIL totals. `health` summarizes rack quality metrics.

Validation checks include required metadata, safe plate names, valid ingredient names, Go template parsing, undeclared template variables, unused ingredients, duplicate tags, category naming, and duplicate plate names across categories.

Example passing output:

```text
PASS

No issues found.
```

Example rack health output:

```text
Rack Health

Total Plates: 7
Categories: 4
Tags: 18

Passing: 7
Warnings: 0
Failing: 0

Duplicate Names: 0
Unused Ingredients: 0
Undeclared Variables: 0
```

## Output Management

Every successful `stamp` or `render` creates a render record in:

```text
data/output/history.json
```

Saved rendered text goes under `data/output/rendered/`. Exports go under `data/output/exports/`.

```text
use scanning/nmap_full_tcp
stamp
copy
save output
save output my_scan.txt
output history
output show 1
output repeat 1
output stats
export markdown
export json
export yaml
```

After a successful render, PLATES prints the stored render ID:

```text
--- Rendered Plate: scanning/nmap_full_tcp ---

sudo nmap -p- --min-rate 5000 -oA C:\Users\USER\code\boxes\devhub/nmap/full_tcp 10.129.202.242

--- End ---

Stored as Render #1
```

`copy` copies the most recent render to the clipboard. On Windows, PLATES uses the built-in `clip` command. On macOS it uses `pbcopy`; on Linux it looks for `xclip` or `xsel`.

`output clear` asks for confirmation before deleting history:

```text
output clear
Type YES to continue:
YES
```

## Polish And UX

PLATES shows a startup banner by default:

```text
██████╗ ██╗      █████╗ ████████╗███████╗███████╗
██╔══██╗██║     ██╔══██╗╚══██╔══╝██╔════╝██╔════╝
██████╔╝██║     ███████║   ██║   █████╗  ███████╗
██╔═══╝ ██║     ██╔══██║   ██║   ██╔══╝  ╚════██║
██║     ███████╗██║  ██║   ██║   ███████╗███████║
╚═╝     ╚══════╝╚═╝  ╚═╝   ╚═╝   ╚══════╝╚══════╝

      Plate-Based Command Rendering System
Type 'help' for commands or 'guide' for built-in guides.
```

Configuration is stored in:

```text
data/config.yaml
```

Default config:

```yaml
banner: true
theme: default
prompt_style: full
tips: true
```

Config commands:

```text
config show
config set banner false
config set banner true
config set theme default
config set theme minimal
config set prompt_style full
config set prompt_style compact
config set tips true
config set tips false
config set store_secret_outputs false
```

Prompt styles:

```text
full     PLATES[devhub][scanning/nmap_full_tcp] >
compact  PLATES >
```

Daily-driver helpers:

```text
tip
fortune
random plate
random plate --use
version
about
```

Autocomplete is enabled for common command trees, guide topics, config keys and values, rack plate names, and categories when the rack can be indexed at shell startup.

## Secrets And Environment Variables

Secrets are stored separately from pantry and workspace values:

```text
data/secrets/secrets.yaml
```

This MVP secret store is local plaintext. It is ignored by git through `.gitignore`, but you should still treat it carefully.

```text
secret set password SuperSecret123
secret get password
secret reveal password
secret list
secret delete password
secret clear --force
```

Template syntax:

```text
{{secret "password"}}
{{env "HOME"}}
{{env "PLATES_TOKEN"}}
```

Secret ingredients can be marked in YAML:

```yaml
ingredients:
  password:
    description: Password for the target service
    required: true
    secret: true
```

Secret values are masked in `show pantry`, `show workspace`, and `show ingredients` when the selected plate marks that ingredient as secret.

By default, renders containing secrets store redacted output in history and exports. The live `copy` command can still copy the most recent raw render immediately after stamping. To allow raw secret output in history, explicitly run:

```text
config set store_secret_outputs true
```

## Plate Packs

Plate Packs are shareable `.zip` archives containing a manifest and selected rack plates:

```text
plates-pack/
|-- pack.yaml
`-- rack/
    |-- scanning/
    |   `-- nmap_full_tcp.yml
    `-- files/
        `-- http_server.yml
```

Packs are stored under:

```text
data/packs/exported/
data/packs/imported/
```

Export examples:

```text
pack export core
pack export scans --category scanning
pack export web-tools --tag web
pack export one-scan --plate scanning/nmap_full_tcp
```

Inspect and validate before importing:

```text
pack inspect data/packs/exported/core.zip
pack validate data/packs/exported/core.zip
```

Import examples:

```text
pack import data/packs/exported/core.zip
pack import data/packs/exported/core.zip --force
```

Default import behavior does not overwrite existing plates. If conflicts exist, PLATES prints the conflicting plate keys and stops. Use `--force` only when you intend to replace local rack files.

Safety notes:

- PLATES never executes pack contents.
- Imports accept only `pack.yaml` and `.yml` / `.yaml` files under `rack/`.
- Absolute paths and path traversal entries are rejected.
- Plates are linted before import.
- Review imported plates before use.

## Example Workflow

```text
PLATES > init
Initialized data directories.
PLATES > workspace devhub
Workspace: devhub
PLATES[devhub] > set target 10.129.202.242
PLATES[devhub] > set workdir C:\Users\USER\code\boxes\devhub
PLATES[devhub] > setg http_port 8000
PLATES[devhub] > list plates
Available Plates:

files/
files/http_server
files/smb_server

notes/
notes/create_notes

scanning/
scanning/nmap_full_tcp
scanning/nmap_services
scanning/rustscan_basic

web/
web/feroxbuster_basic
PLATES[devhub] > use scanning/nmap_full_tcp
Loaded plate: scanning/nmap_full_tcp
PLATES[devhub][scanning/nmap_full_tcp] > show plate
Name: nmap_full_tcp
Category: scanning
Description: Full TCP port scan with output files
Tags: nmap, scanning, tcp
Path: data/rack/scanning/nmap_full_tcp.yml
Ingredients: 3

Template Preview:
  sudo nmap -p- --min-rate {{rate}} -oA {{workdir}}/nmap/full_tcp {{target}}
PLATES[devhub][scanning/nmap_full_tcp] > show ingredients
Required:
  target   Target IP address or hostname       = 10.129.202.242
  workdir  Working directory for output files  = C:\Users\USER\code\boxes\devhub

Optional:
  rate     Minimum packet rate  default: 5000  = 5000
PLATES[devhub][scanning/nmap_full_tcp] > stamp
--- Rendered Plate: scanning/nmap_full_tcp ---

sudo nmap -p- --min-rate 5000 -oA C:\Users\USER\code\boxes\devhub/nmap/full_tcp 10.129.202.242

--- End ---
```

## Browsing Plates

`list plates` groups available plates by category and shows each description. Use `show rack` for a summary of total plates, category counts, and tag counts.

```text
PLATES > list plates
PLATES > show rack
```

## Searching Plates

`search plates <query>` searches plate names, categories, descriptions, tags, ingredient names, and ingredient descriptions. Search is case-insensitive and supports partial matches.

```text
PLATES > search plates nmap
PLATES > search plates http
PLATES > search plates workdir
```

## Tags And Categories

Use `show tags` to inspect tag counts and `show category <name>` to browse one category.

```text
PLATES > show tags
PLATES > show category scanning
PLATES > use scanning/nmap_full_tcp
PLATES[scanning/nmap_full_tcp] > show plate
```

## Design Notes

The shell package owns the interactive loop and translates user input into operations. Persistence is behind `config.VariableStore`. Rack discovery, search, and loading are behind `plates.Browser`, `plates.Discoverer`, and `plates.Loader`. Rendering is behind `plates.Renderer`.

This keeps command parsing, storage, plate loading, variable resolution, and rendering separate so future phases can add validation, forge mode, search, linting, clipboard support, and richer renderers without reshaping the project.
