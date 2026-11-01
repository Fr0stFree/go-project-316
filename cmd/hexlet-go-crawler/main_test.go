package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"code/internal/report"

	"github.com/stretchr/testify/require"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func testResponse() *http.Response {
	body := "<title>Simple site</title><h1>Welcome</h1>"
	response := &http.Response{
		StatusCode:    http.StatusOK,
		Header:        make(http.Header),
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
	}
	response.Header.Set("Content-Type", "text/html")

	return response
}

func TestCLIProcess(_ *testing.T) {
	if os.Getenv("CRAWLER_TEST_PROCESS") != "1" {
		return
	}

	separator := slices.Index(os.Args, "--")
	os.Args = append([]string{"hexlet-go-crawler"}, os.Args[separator+1:]...)

	http.DefaultTransport = roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		if request.URL.String() != "http://example.test" ||
			request.Method != http.MethodGet ||
			request.Header.Get("User-Agent") != "TestCLI/1.0" {
			return nil, fmt.Errorf("unexpected request: %s %s", request.Method, request.URL)
		}

		return testResponse(), nil
	})

	main()
	os.Exit(0)
}

func runCLI(t *testing.T, args ...string) ([]byte, []byte, int) {
	t.Helper()

	commandArgs := append([]string{"-test.run=^TestCLIProcess$", "--"}, args...)
	command := exec.CommandContext(t.Context(), os.Args[0], commandArgs...)

	command.Env = append(os.Environ(), "CRAWLER_TEST_PROCESS=1")

	var stdout bytes.Buffer

	var stderr bytes.Buffer

	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	if err == nil {
		return stdout.Bytes(), stderr.Bytes(), 0
	}

	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)

	return stdout.Bytes(), stderr.Bytes(), exitErr.ExitCode()
}

func TestCLI(t *testing.T) {
	args := []string{
		"--depth", "1",
		"--retries", "1",
		"--timeout", "2s",
		"--user-agent", "TestCLI/1.0",
		"--workers", "1",
		"http://example.test",
	}

	for _, test := range []struct {
		name    string
		verbose bool
	}{
		{name: "success"},
		{name: "verbose output", verbose: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := append([]string(nil), args...)
			if test.verbose {
				input = append([]string{"--verbose"}, input...)
			}

			start := time.Now()
			stdout, stderr, exitCode := runCLI(t, input...)

			require.Zero(t, exitCode)
			require.True(t, bytes.HasSuffix(stdout, []byte("\n")))
			require.False(t, bytes.HasSuffix(stdout, []byte("\n\n")))

			var actual report.Report
			require.NoError(t, json.Unmarshal(stdout, &actual))
			require.Len(t, actual.Pages, 1)
			require.WithinDuration(t, start, actual.GeneratedAt, 5*time.Second)
			require.WithinDuration(t, start, actual.Pages[0].DiscoveredAt, 5*time.Second)

			expected := report.Report{
				RootURL:     "http://example.test",
				Depth:       1,
				GeneratedAt: actual.GeneratedAt,
				Pages: []report.Page{{
					URL:          "http://example.test",
					HTTPStatus:   http.StatusOK,
					Status:       "ok",
					SEO:          report.SEO{HasTitle: true, Title: "Simple site", HasH1: true},
					BrokenLinks:  []report.NodeLink{},
					Assets:       []report.Asset{},
					DiscoveredAt: actual.Pages[0].DiscoveredAt,
				}},
			}
			require.Equal(t, expected, actual)

			expectedJSON, err := json.MarshalIndent(expected, "", "  ")
			require.NoError(t, err)

			require.Equal(t, string(append(expectedJSON, '\n')), string(stdout))

			if test.verbose {
				require.Contains(t, string(stderr), "level=DEBUG")
				require.Contains(t, string(stderr), "Making HTTP request")
			} else {
				require.Empty(t, stderr)
			}
		})
	}

	t.Run("invalid URL", func(t *testing.T) {
		stdout, stderr, exitCode := runCLI(t, "ftp://example.test")

		require.Equal(t, 1, exitCode)
		require.Empty(t, stdout)
		require.Equal(t, "something went wrong during analysis of `ftp://example.test`: unsupported root URL scheme \"ftp\": only http and https are allowed\n", string(stderr))
	})

	t.Run("missing URL", func(t *testing.T) {
		stdout, stderr, exitCode := runCLI(t)

		require.Equal(t, 1, exitCode)
		require.Empty(t, stdout)
		require.Equal(t, "expected 1 argument, got 0\n", string(stderr))
	})

	t.Run("invalid workers flag", func(t *testing.T) {
		stdout, stderr, exitCode := runCLI(t, "--workers", "-1", "http://example.test")

		require.Equal(t, 1, exitCode)
		require.Empty(t, stdout)
		require.Equal(t, "something went wrong during analysis of `http://example.test`: size cannot be negative\n", string(stderr))
	})
}
