# Forge Mode

Forge Mode is the interactive plate creation workflow. It builds YAML plate templates from inside the PLATES shell and never executes draft content.

Start Forge Mode:

```text
forge
use forge
```

## Commands

```text
set name <value>
set category <value>
set description <value>
add_line <text>
insert_line <number> <text>
delete_line <number>
clear_lines
add_var <name> <description>
add_secret_var <name> <description>
add_optional_var <name> <default> <description>
set_var_required <name> <true|false>
set_var_default <name> <value>
rm_var <name>
add_tag <tag>
rm_tag <tag>
show lines
show vars
show tags
show draft
validate
save
save --force
cancel
```

## Complete Example

```text
PLATES > forge
FORGE[new] > set name nxc_ldap_commands
FORGE[nxc_ldap_commands] > set category ldap
FORGE[nxc_ldap_commands] > set description "Common NetExec LDAP command helper"
FORGE[nxc_ldap_commands] > add_line "# LDAP command helper for {{target}}"
FORGE[nxc_ldap_commands] > add_line "nxc ldap {{target}} -u {{username}} -p {{password}}"
FORGE[nxc_ldap_commands] > add_var target "Target IP address or hostname"
FORGE[nxc_ldap_commands] > add_var username "Username"
FORGE[nxc_ldap_commands] > add_var password "Password or quoted empty string"
FORGE[nxc_ldap_commands] > add_tag nxc
FORGE[nxc_ldap_commands] > add_tag ldap
FORGE[nxc_ldap_commands] > show draft
FORGE[nxc_ldap_commands] > validate
FORGE[nxc_ldap_commands] > save
PLATES >
```

## Saving

Forge saves to:

```text
data/rack/<category>/<name>.yml
```

Existing files are not overwritten by default. Use `save --force` only when you intentionally want to replace an existing plate.
