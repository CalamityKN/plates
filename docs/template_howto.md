# Plate Template How-To

A plate is a reusable command template. PLATES loads plates from YAML files, resolves variables called ingredients, and renders command text for manual review and copying. PLATES does not execute commands.

## Location

Store plate files under:

```text
data/rack/
```

Use category directories to organize them:

```text
data/rack/scanning/nmap_full_tcp.yml
data/rack/files/http_server.yml
```

## YAML Schema

```yaml
name: string
category: string
description: string
tags:
  - string
ingredients:
  variable_name:
    description: string
    required: true
    secret: optional bool
    default: optional string
template: |
  command text with {{variable_name}}
```

Required fields are `name`, `category`, `description`, and `template`. `tags` and `ingredients` are optional, though most useful plates define ingredients.

## Ingredients

Ingredients are variables used by the template. Required ingredients must resolve before `stamp` or `render` prints output. Optional ingredients can include defaults.

```yaml
ingredients:
  target:
    description: Target IP address or hostname
    required: true
  rate:
    description: Minimum packet rate
    required: false
    default: "5000"
```

## Template Syntax

Use simple Go template variables:

```text
{{target}}
{{workdir}}
{{http_port}}
```

Variable names must use letters, numbers, and underscores, and must not start with a number.

Secret and environment helpers:

```text
{{secret "password"}}
{{env "HOME"}}
```

## Core Example

```yaml
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
```

## From Scratch

1. Pick a category such as `scanning`, `files`, or `web`.
2. Create a YAML file under `data/rack/<category>/`.
3. Add metadata: name, category, description, and tags.
4. Define ingredients for every variable in the template.
5. Add the template block.
6. Run `list plates`, `use <category>/<name>`, `show ingredients`, and `stamp`.

## Common Mistakes

- Missing required fields.
- Using `{{.target}}` instead of `{{target}}`.
- Referencing a variable in the template without adding an ingredient.
- Forgetting `secret: true` for sensitive ingredients such as passwords or tokens.
- Using spaces or punctuation in ingredient names.
- Duplicating plate names across categories and then using only the short name.

## Validation Expectations

PLATES validates required metadata, ingredient names, and template parsing. Missing required values are detected when stamping.
