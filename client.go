package rtorrent

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

const DefaultTimeout = 6 * time.Second

// Client is an XML-RPC client for rTorrent, it is safe for concurrent use.
type Client struct {
	t       transport
	timeout time.Duration
}

// Option configures a Client constructed by Dial, DialUnix, or DialHTTP.
type Option func(*Client)

func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.timeout = d
	}
}

// WithBasicAuth sets HTTP Basic Auth credentials on requests made by a
// Client constructed with DialHTTP.
func WithBasicAuth(username, password string) Option {
	return func(c *Client) {
		if t, ok := c.t.(*httpTransport); ok {
			t.username = username
			t.password = password
		}
	}
}

// Dial returns a Client that connects directly to a SCGI listener
// over TCP at addr.
func Dial(addr string, opts ...Option) *Client {
	return newClient(&scgiTransport{network: "tcp", address: addr}, opts)
}

// DialUnix returns a Client that connects directly to a SCGI
// listener over a Unix domain socket at path.
func DialUnix(path string, opts ...Option) *Client {
	return newClient(&scgiTransport{network: "unix", address: path}, opts)
}

// DialHTTP returns a Client that sends XML-RPC requests to url over HTTP,
// for setups where a SCGI listener is proxied by a web server (e.g. nginx).
func DialHTTP(url string, opts ...Option) *Client {
	return newClient(&httpTransport{url: url, httpClient: &http.Client{}}, opts)
}

// newClient applies opts over a Client wrapping t.
func newClient(t transport, opts []Option) *Client {
	c := &Client{t: t, timeout: DefaultTimeout}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Close releases resources held by Client.
func (c *Client) Close() error {
	return nil
}

// Call invokes the XML-RPC method name with params and returns its single
// return value.
func (c *Client) Call(ctx context.Context, name string, params ...Value) (Value, error) {
	if _, ok := ctx.Deadline(); !ok && c.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.timeout)
		defer cancel()
	}

	reqBody := encodeMethodCall(name, params)

	respBody, err := c.t.call(ctx, reqBody)
	if err != nil {
		return Value{}, fmt.Errorf("rtorrent: call %s: %w", name, err)
	}

	v, err := decodeMethodResponse(respBody)
	if err != nil {
		return Value{}, fmt.Errorf("rtorrent: call %s: %w", name, err)
	}
	return v, nil
}

// LoadRaw loads a torrent directly from its raw bencoded structure.
func (c *Client) LoadRaw(ctx context.Context, data []byte) error {
	_, err := c.Call(ctx, "load.raw", NewString(""), NewBase64(data))
	return err
}

// LoadRawStart loads a torrent directly from its raw bencoded structure and
// starts it immediately.
func (c *Client) LoadRawStart(ctx context.Context, data []byte) error {
	_, err := c.Call(ctx, "load.raw_start", NewString(""), NewBase64(data))
	return err
}

// LoadStart loads a torrent from a filesystem path (or URL) by
// and starts it immediately.
func (c *Client) LoadStart(ctx context.Context, path string) error {
	_, err := c.Call(ctx, "load.start", NewString(""), NewString(path))
	return err
}

// Multicall invokes an rTorrent multicall method (e.g. "d.multicall2",
// "t.multicall", "p.multicall", "f.multicall") and returns one row of
// Values per result, one column per command in cmds.
func (c *Client) Multicall(ctx context.Context, method string, leading []Value, cmds ...string) ([][]Value, error) {
	params := make([]Value, 0, len(leading)+len(cmds))
	params = append(params, leading...)
	for _, cmd := range cmds {
		params = append(params, NewString(cmd))
	}

	v, err := c.Call(ctx, method, params...)
	if err != nil {
		return nil, err
	}
	return multicallRows(v, len(cmds))
}

// multicallRows converts a multicall response (an array of one row per
// result, each row an array with one element per requested command) into
// [][]Value.
func multicallRows(v Value, wantCols int) ([][]Value, error) {
	entries, err := v.AsArray()
	if err != nil {
		return nil, fmt.Errorf("rtorrent: multicall response: %w", err)
	}

	rows := make([][]Value, len(entries))
	for i, entry := range entries {
		row, err := entry.AsArray()
		if err != nil {
			return nil, fmt.Errorf("rtorrent: multicall response: row %d: %w", i, err)
		}
		if len(row) != wantCols {
			return nil, fmt.Errorf("rtorrent: multicall response: row %d: got %d columns, want %d", i, len(row), wantCols)
		}
		rows[i] = row
	}
	return rows, nil
}
