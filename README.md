# Monzter

Monzter ...

- is an HTML link crawler with configurable maximum depth and rate limiting
- crawls sites depth-first
- will only visit pages located on the same hostname as the entrypoint (links to other hostnames will still be listed in the output)

## Installation

	go get github.com/sauerbraten/monzter

## Usage

	monzter [-depth=<D>] [-rate=<R>] <entrypoint>

`D` is the maximum depth of pages to crawl (defaults to 1) and `R` is the maximum number of requests per second (defaults to 10); `entrypoint` is the page to start the crawling on and must be an absolute URL.

## Example

	monzter -depth 2 -rate 20 https://news.ycombinator.com/