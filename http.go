package rtorrent

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
)

const maxHTTPResponseBytes = 8 << 20

type httpTransport struct {
	url        string
	httpClient *http.Client
}

// call implements transport.
func (t *httpTransport) call(ctx context.Context, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("http: build request: %w", err)
	}
	req.Header.Set("Content-Type", "text/xml")

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: do request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxHTTPResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("http: read response body: %w", err)
	}
	if len(respBody) > maxHTTPResponseBytes {
		return nil, fmt.Errorf("http: response body exceeds %d bytes", maxHTTPResponseBytes)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http: unexpected status %d: %s", resp.StatusCode, bodyPrefix(respBody))
	}

	return respBody, nil
}

func bodyPrefix(b []byte) []byte {
	const maxPrefix = 256
	if len(b) > maxPrefix {
		return b[:maxPrefix]
	}
	return b
}
