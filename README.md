# Hexlet Crawler

Hexlet Crawler is a Go command-line tool for scanning a website and producing a JSON report. It follows links on the starting URL's hostname, checks internal and external links for failures, collects image, script, and stylesheet metadata, and extracts basic SEO information from each page. The crawler is also available as a Go package.

## Features

- Crawls pages on the starting URL's hostname up to a configurable depth.
- Checks links outside that hostname without crawling their pages and reports broken links on the pages that reference them.
- Reports HTTP status and request errors for crawled pages.
- Collects the URL, type, response status, and size of referenced images, scripts, and stylesheets.
- Extracts the page title, meta description, and presence of an H1 heading.
- Supports request timeouts, retry attempts, concurrency, a custom User-Agent, and either a delay or a requests-per-second limit.
- Writes the report as JSON to standard output; `--verbose` writes debug logs to standard error.
- Exposes `crawler.Analyze` for use from Go code.

## Pipeline Status

[![Hexlet check](https://github.com/Fr0stFree/go-project-316/actions/workflows/hexlet-check.yml/badge.svg)](https://github.com/Fr0stFree/go-project-316/actions/workflows/hexlet-check.yml)
[![Tests and linters](https://github.com/Fr0stFree/go-project-316/actions/workflows/test-and-lint.yml/badge.svg)](https://github.com/Fr0stFree/go-project-316/actions/workflows/test-and-lint.yml)

## Requirements

- Go `1.26.3`
- `make` for the development commands below

The lint and formatting targets use `golangci-lint` `v2.12.2`. Install it once with `make install-lint` if you plan to run those targets.

## Local Setup

Clone the repository and download the Go modules:

```bash
git clone https://github.com/Fr0stFree/go-project-316.git
cd go-project-316
make install
```

Build the executable:

```bash
make build
```

The binary is written to `bin/crawler`. No database or configuration file is required.

## Usage

Pass an HTTP or HTTPS URL to the binary:

```bash
./bin/crawler https://example.com
```

You can run the same entry point without building the binary first:

```bash
go run ./cmd/hexlet-go-crawler https://example.com
```

The Makefile also builds and runs the binary:

```bash
make run URL=https://example.com ARGS="--depth 2"
```

Set the crawl depth and request limits with flags before the URL:

```bash
go run ./cmd/hexlet-go-crawler --depth 2 --retries 2 --delay 200ms --timeout 10s https://example.com
```

`--depth 1` includes only the starting page in `pages`; its links are still checked for failures. The default depth is `10`. `--retries N` allows one initial request and up to N additional attempts for temporary failures such as 429, 5xx, or retryable network errors. The default is `1`, so temporary failures get one additional attempt.

Links on the starting URL's hostname are followed until the depth limit. At the limit, and for links on other hostnames, the crawler checks the target with `HEAD` but does not add it to `pages`. If the server rejects `HEAD` with 405 or 501, the crawler tries `GET`. A failed check appears in the referring page's `broken_links`. Referenced assets can be fetched from other hosts; when `HEAD` is unsupported or does not provide a size, the crawler uses `GET` to measure the response body.

Hostnames are compared without regard to case, and URL fragments do not create duplicate visits. Responses with a `text/html`, `application/xml`, or `text/xml` content type are parsed for links and SEO fields. Text in titles, descriptions, and H1 headings is normalized for whitespace; the report records whether each field was present.

| Flag | Description | Default |
| --- | --- | --- |
| `--depth` | Number of page levels to include, starting with the root page | `10` |
| `--retries` | Additional attempts after the first request | `1` |
| `--delay` | Minimum time between requests, using Go duration syntax | `0s` |
| `--timeout` | Timeout for each HTTP request | `15s` |
| `--rps` | Maximum requests per second; takes precedence over `--delay` | `0` (disabled) |
| `--user-agent` | User-Agent sent with HTTP requests | `HexletCrawler/1.0` |
| `--workers` | Concurrent crawl workers | `4` |
| `--verbose` | Write debug logs to standard error | `false` |

For example, redirect the JSON report to a file while keeping debug logs visible in the terminal:

```bash
./bin/crawler --verbose --depth 2 https://example.com > report.json
```

Run `./bin/crawler --help` for the complete CLI help.

## JSON Report

The top level contains `root_url`, the requested `depth`, a UTC `generated_at` timestamp, and a URL-sorted `pages` array. Each page includes its URL, depth, HTTP status, `ok` or `error` state, discovery time, SEO fields, broken links, and assets. Pages and assets include `error` only when there is an error message. An HTTP failure can have `status: "error"` without an `error` field. Empty `broken_links` and `assets` fields are JSON arrays after a completed request, and `null` when the page request itself failed.

HTTP errors and failed requests to valid HTTP(S) URLs are recorded in the report. Invalid root URLs cause the CLI to exit with a nonzero status and write the error to standard error.

This example shows the report for a single-page test site:

```json
{
  "root_url": "http://example.test",
  "depth": 1,
  "generated_at": "2024-01-02T03:04:05Z",
  "pages": [
    {
      "url": "http://example.test",
      "depth": 0,
      "http_status": 200,
      "status": "ok",
      "seo": {
        "has_title": true,
        "title": "Simple site",
        "has_description": false,
        "description": "",
        "has_h1": true
      },
      "broken_links": [],
      "assets": [],
      "discovered_at": "2024-01-02T03:04:05Z"
    }
  ]
}
```

The example uses a fixed test clock. Real timestamps and page details depend on the site and run time.

## Development Commands

| Command | Description |
| --- | --- |
| `make install` | Download Go modules. |
| `make build` | Build `bin/crawler` from `cmd/hexlet-go-crawler`. |
| `make start URL=... ARGS="..."` | Build and run the CLI with the supplied URL and optional flags. |
| `make run URL=... ARGS="..."` | Build and run the CLI with the supplied URL and optional flags. |
| `make test` | Run the repository's Go tests with verbose output. |
| `go test -race ./...` | Run all tests with the race detector, as in CI. |
| `make test-coverage` | Write `coverage.out` and print coverage by function. |
| `make install-lint` | Install the pinned `golangci-lint` version. |
| `make lint` | Run the configured linters. |
| `make fmt` | Format code through `golangci-lint fmt`. |
| `make fmt-check` | Check formatting without changing files. |
| `make tidy-check` | Check whether the Go module files are tidy. |
| `make lint-fix` | Format code and apply supported lint fixes. |

Typical checks before pushing:

```bash
make build
make test
go test -race ./...
make fmt-check
make tidy-check
make lint
```

The repository has a test-and-lint workflow that also runs the race detector and generates a coverage report, plus a separate Hexlet project check. The Hexlet workflow runs the course's integration tests on pushes.

## Project Structure

```text
.
|-- cmd/hexlet-go-crawler  # CLI entry point
|-- crawler                # Public Analyze API and options
|-- internal/cliapp        # CLI flags and output
|-- internal/crawler       # Fetching, parsing, rate limiting, and workers
|-- internal/report        # JSON report types and assembly
|-- .github/workflows      # CI and Hexlet checks
|-- Makefile               # Development commands
`-- README.md              # Project documentation
```
