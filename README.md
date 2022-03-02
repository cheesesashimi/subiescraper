# subiescraper

This is a toy project I built to scrape Subaru dealer inventory in the USA. For
the most part, the code isn't in the best shape because there's a lot of
proof-of-concept stuff everywhere. However, it does (mostly) work!

## How to use

The part you're probably interested in is located in the `cmd/subiescraper` folder. To use, you must have a Golang toolchain installed:

1. Clone this repo: `$ git clone https://github.com/cheesesashimi/subiescraper.git`
2. `$ cd cmd/subiescraper`
3. `$ go build .`
4. `$ ./subiescraper --help`

A future TODO is to have CI running on this repo publishing ready-to-use binaries.
