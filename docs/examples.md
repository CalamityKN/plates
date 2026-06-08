# Examples

## Initialize PLATES

```text
init
```

## Create A Workspace

```text
workspace devhub
set target 10.129.202.242
set workdir C:\Users\knjoh\code\boxes\devhub
```

## Set Pantry Values

```text
setg my_ip 10.10.14.3
setg http_port 8000
show pantry
```

## Use A Scanning Plate

```text
list plates
search plates nmap
use scanning/nmap_full_tcp
show plate
show ingredients
stamp
```

## Use A File-Serving Plate

```text
use files/http_server
show ingredients
render
```

## Search Plates

```text
search plates http
search plates workdir
show category files
show tags
```

## Create A Plate With Forge Mode

```text
forge
set name quick_ping
set category misc
set description "Render a quick ping command"
add_line "ping -c 4 {{target}}"
add_var target "Target IP address or hostname"
add_tag ping
show draft
validate
save
use misc/quick_ping
```
