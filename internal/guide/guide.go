package guide

import (
	"fmt"
	"strings"
)

type Topic struct {
	Name        string
	Description string
	Text        string
}

var topics = []Topic{
	{
		Name:        "plates",
		Description: "How to build and use plate YAML files",
		Text: `Plate YAML Guide

A plate is a reusable command template stored under data/rack/.
PLATES loads YAML plates, resolves ingredients, and renders text only.

Required fields:
  name          File-safe plate name
  category      Rack category, usually matching the directory
  description   Human-readable summary
  template      Command template text

Common optional fields:
  tags          Searchable labels
  ingredients   Variables used by the template

Example:
  name: http_server
  category: files
  description: Start a local Python HTTP server from the working directory
  tags:
    - python
    - http
    - files
  ingredients:
    workdir:
      description: Directory to serve files from
      required: true
    http_port:
      description: Local HTTP server port
      required: false
      default: "8000"
  template: |
    cd {{workdir}}
    python3 -m http.server {{http_port}}

Template variables use simple names like {{target}} or {{http_port}}.
Required ingredients must resolve before stamp/render will print output.
Optional ingredients can provide defaults.`,
	},
	{
		Name:        "forge",
		Description: "Interactive plate creation with Forge Mode",
		Text: `Forge Mode Guide

Start Forge Mode:
  forge
  use forge

Core commands:
  set name <value>
  set category <value>
  set description <value>
  add_line <text>
  insert_line <number> <text>
  delete_line <number>
  clear_lines
  add_var <name> <description>
  add_optional_var <name> <default> <description>
  add_tag <tag>
  show draft
  validate
  save
  save --force
  cancel

Example:
  forge
  set name my_plate
  set category scanning
  set description "My reusable command"
  add_line "tool --target {{target}}"
  add_var target "Target IP address"
  add_tag scanning
  validate
  save

Saving writes data/rack/<category>/<name>.yml.
Existing files require save --force to overwrite.`,
	},
	{
		Name:        "variables",
		Description: "Pantry, workspace variables, ingredients, and defaults",
		Text: `Variables Guide

PLATES resolves values from highest to lowest priority:
  1. Session/workspace values set with set during the active session
  2. Workspace YAML values from data/workspaces/<workspace>.yaml
  3. Pantry/global values from data/pantry/globals.yaml
  4. Ingredient defaults from the plate YAML

Commands:
  setg <key> <value>       Set a global pantry value
  set <key> <value>        Set an active workspace value
  secret set <key> <value> Store a local secret
  secret list              List masked secrets
  show pantry              Show global values
  show workspace           Show active workspace values
  show ingredients         Show required and optional plate variables

Missing required ingredients prevent stamp/render.
Defaults are useful for common options such as http_port or rate.

Secret syntax:
  {{secret "password"}}

Environment syntax:
  {{env "HOME"}}

Secrets are masked in normal displays. Local secrets are stored in
data/secrets/secrets.yaml as plaintext unless future encryption is added.`,
	},
	{
		Name:        "rack",
		Description: "How to organize and search the plate rack",
		Text: `Rack Guide

The rack root is data/rack.
Categories are directories, such as:
  scanning
  files
  web
  notes
  windows
  linux
  ldap
  database
  cloud
  misc

Use descriptive names and avoid duplicate plate names when possible.
Use tags for tools, protocols, platforms, and workflow types.

Commands:
  list plates
  search plates <query>
  show rack
  show tags
  show category <name>

Search checks names, categories, descriptions, tags, ingredient names,
and ingredient descriptions.`,
	},
	{
		Name:        "examples",
		Description: "Common PLATES workflows",
		Text: `Examples Guide

Initialize and create a workspace:
  init
  workspace devhub

Set values:
  set target 10.129.202.242
  set workdir C:\Users\knjoh\code\boxes\devhub
  setg http_port 8000

Browse and use a scanning plate:
  list plates
  search plates nmap
  use scanning/nmap_full_tcp
  show ingredients
  stamp

Use a file-serving plate:
  use files/http_server
  show ingredients
  render

Create a plate:
  forge
  set name my_plate
  set category misc
  add_line "tool {{target}}"
  add_var target "Target host"
  validate
  save`,
	},
	{
		Name:        "safety",
		Description: "Renderer-only design and safe usage notes",
		Text: `Safety Guide

PLATES renders commands only.
PLATES does not execute rendered commands or Forge draft content.

Review output before copying commands into a terminal.
Store secrets carefully. Pantry and workspace YAML files are plain text.
Avoid saving sensitive passwords in workspace files when possible.
Use secret set <key> <value> for values you do not want displayed normally.
The local secret store is also plaintext, but it is kept separate and ignored
by git by default.

Future phases may add secret-store support, but today PLATES is a
local renderer and organizer for command templates.

Rendered history stores redacted output for secret-bearing renders unless
store_secret_outputs is explicitly enabled in config.`,
	},
}

func List() string {
	var b strings.Builder
	b.WriteString("Available guides:\n\n")
	for _, topic := range topics {
		fmt.Fprintf(&b, "  %-10s %s\n", topic.Name, topic.Description)
	}
	b.WriteString("\nUse: guide <topic>\n")
	return b.String()
}

func Show(name string) (string, error) {
	for _, topic := range topics {
		if topic.Name == name {
			return topic.Text + "\n", nil
		}
	}
	return "", fmt.Errorf("unknown guide topic %q; run 'guide' for available topics", name)
}
