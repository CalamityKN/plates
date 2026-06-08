# Rack Organization

The rack root is:

```text
data/rack
```

Categories are directories under the rack root. A plate in `data/rack/scanning/nmap_full_tcp.yml` has category `scanning`.

## Recommended Categories

- scanning
- files
- web
- notes
- windows
- linux
- ldap
- database
- cloud
- misc

Slash-separated subcategories are supported by Forge for category paths such as `windows/ldap`.

## Naming

Use concise names with letters, numbers, underscores, or hyphens. Avoid duplicate plate names when possible. If duplicates exist, use `category/name` when loading a plate.

## Tags

Tags help search and browsing. Good tags include tool names, protocols, platforms, and workflow types.

```yaml
tags:
  - nmap
  - scanning
  - tcp
```

## Searching

```text
list plates
search plates nmap
show rack
show tags
show category scanning
```

Search checks plate names, categories, descriptions, tags, ingredient names, and ingredient descriptions.
