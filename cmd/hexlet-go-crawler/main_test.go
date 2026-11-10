package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
		if os.Getenv("CRAWLER_TEST_SIGINT") == "1" {
			switch request.URL.Path {
			case "":
				body := `<a href="/slow">Slow page</a>`
				response := testResponse()
				response.Body = io.NopCloser(strings.NewReader(body))
				response.ContentLength = int64(len(body))

				return response, nil

			case "/slow":
				readyFile := os.Getenv("CRAWLER_TEST_READY_FILE")
				if err := os.WriteFile(readyFile, []byte("ready"), 0600); err != nil {
					return nil, err
				}

				<-request.Context().Done()

				return nil, request.Context().Err()
			}
		}

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
	t.Run("should return report", func(t *testing.T) {
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
			{name: "successfully"},
			{name: "with verbose output", verbose: true},
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
	})

	t.Run("should return partial report on SIGINT", func(t *testing.T) {
		readyFile := filepath.Join(t.TempDir(), "ready")

		args := []string{
			"-test.run=^TestCLIProcess$",
			"--",
			"--depth", "2",
			"--workers", "1",
			"--user-agent", "TestCLI/1.0",
			"http://example.test",
		}

		cmd := exec.CommandContext(t.Context(), os.Args[0], args...)

		cmd.Env = append(
			os.Environ(),
			"CRAWLER_TEST_PROCESS=1",
			"CRAWLER_TEST_SIGINT=1",
			"CRAWLER_TEST_READY_FILE="+readyFile,
		)

		var stdout, stderr bytes.Buffer

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		require.NoError(t, cmd.Start())
		t.Cleanup(func() { _ = cmd.Process.Kill() })

		deadline := time.After(5 * time.Second)

		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			if _, err := os.Stat(readyFile); err == nil {
				break
			}

			select {
			case <-ticker.C:
			case <-deadline:
				t.Fatal("slow HTTP request was not started")
			}
		}

		require.NoError(t, cmd.Process.Signal(os.Interrupt))

		err := cmd.Wait()

		var exitErr *exec.ExitError
		require.ErrorAs(t, err, &exitErr)
		require.Equal(t, 1, exitErr.ExitCode())

		var actual report.Report
		require.NoError(t, json.Unmarshal(stdout.Bytes(), &actual))

		require.Equal(t, "http://example.test", actual.RootURL)
		require.Len(t, actual.Pages, 1)
		require.Equal(t, "http://example.test", actual.Pages[0].URL)
		require.Equal(t, http.StatusOK, actual.Pages[0].HTTPStatus)
		require.Equal(t, "crawl canceled: partial report generated\n", stderr.String())
	})

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
		require.Contains(t, string(stdout), "USAGE:")
		require.Contains(t, string(stderr), "--workers must be greater than zero")
	})
}
