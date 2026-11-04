package httpfetcher

import "net/http"

type rateLimitedTransport struct {
	http.RoundTripper
	fetcher *Fetcher
}

func (t *rateLimitedTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if err := t.fetcher.limiter.Wait(request.Context()); err != nil {
		return nil, err
	}

	return t.RoundTripper.RoundTrip(request)
}
