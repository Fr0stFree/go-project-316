# Hexlet Crawler

Hexlet Crawler is a Go command-line tool for scanning a website and producing a JSON report. It follows links within the starting site's domain, checks for broken links, collects image, script, and stylesheet metadata, and extracts basic SEO information from each page. The crawler is also available as a Go package.

## Features

- Crawls pages on the starting site's domain up to a configurable depth.
- Reports HTTP status, failed requests, and broken links for each page.
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

Set the crawl depth and request limits with flags before the URL:

```bash
go run ./cmd/hexlet-go-crawler --depth 2 --retries 2 --delay 200ms --timeout 10s https://example.com
```

`--depth 1` includes only the starting page in `pages`; its links are still checked for failures. The default depth is `10`. `--retries N` allows one initial request and up to N additional requests. The default `1` allows at most two requests.

Page links are followed and checked when their hostname matches the starting URL. External page links are skipped; referenced assets can still be fetched from other hosts.

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

The top level contains `root_url`, the requested `depth`, a UTC `generated_at` timestamp, and a URL-sorted `pages` array. Each page includes its URL, depth, HTTP status, `ok` or `error` state, discovery time, SEO fields, broken links, and assets. Failed requests also include an `error` message. Successful assets omit the `error` field.

```json
{
  "root_url": "https://example.com",
  "depth": 1,
  "generated_at": "2026-10-18T12:00:00Z",
  "pages": [
    {
      "url": "https://example.com",
      "depth": 0,
      "http_status": 200,
      "status": "ok",
      "seo": {
        "has_title": true,
        "title": "Example Domain",
        "has_description": false,
        "description": "",
        "has_h1": true
      },
      "broken_links": [],
      "assets": [],
      "discovered_at": "2026-10-18T12:00:00Z"
    }
  ]
}
```

The timestamps and page details above illustrate the response format; actual values depend on the site and run time.

## Development Commands

| Command | Description |
| --- | --- |
| `make install` | Download Go modules. |
| `make build` | Build `bin/crawler` from `cmd/hexlet-go-crawler`. |
| `make start ARGS="..."` | Build and run the CLI with the supplied arguments. |
| `make run ARGS="..."` | Run the existing binary with the supplied arguments. |
| `make test` | Run the repository's Go tests with verbose output. |
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
make fmt-check
make tidy-check
make lint
```

The repository has a regular test-and-lint workflow and a separate Hexlet project check. The Hexlet workflow runs the course's integration tests on pushes.

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
