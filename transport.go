package rtorrent

import "context"

// transport sends a single XML-RPC request body over the wire and returns
// the raw XML-RPC response body, with response framing (SCGI headers, HTTP
// status/headers) already stripped.
type transport interface {
	call(ctx context.Context, body []byte) ([]byte, error)
}
