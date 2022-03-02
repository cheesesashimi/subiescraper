# subiescraper

This is the main executable of the program. At present it accepts the following options:

```
NAME:
   subiescraper - Scrape Subaru dealer inventory in North America

USAGE:
   subiescraper [global options]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --state value  What states to scrape, can be combined: --state PA --state OH
   --json         Write output to JSON file by state (data-<state>.json) (default: false)
   --html         Generate an HTML report by state (data-<state>.html) (default: false)
   --help, -h     show help (default: false)
```

The most useful options will be the `--json` and `--html` options. The JSON
output is suitable for consumption with a tool such as
[jq](https://stedolan.github.io/jq/). The HTML is very bare-bones and is
missing a top-level `index.html` page at the moment.

## JSON Output

A common `jq` recipe I use is the following:

```console
$ cat data-*.json | jq -r '.[].new.pageInfo.trackingData[].link'
```

This will yield the URLs of the current inventory in my desired state. If
you're on a Mac, you could add a `| xargs open` to open those in your default
web browser. On a Linux box, you can do `| xargs firefox` to achieve the same
effect.
