package rtorrent

import (
	"context"
	"crypto/tls"
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
//
// It panics if applied to a Client constructed with Dial or DialUnix.
func WithBasicAuth(username, password string) Option {
	return func(c *Client) {
		t, ok := c.t.(*httpTransport)
		if !ok {
			panic("rtorrent: WithBasicAuth used with a non-HTTP transport (Dial/DialUnix); it only applies to DialHTTP")
		}
		t.username = username
		t.password = password
	}
}

// WithTLSConfig sets the TLS client config used for requests made by a
// Client constructed with DialHTTP.
//
// It panics if applied to a Client constructed with Dial or DialUnix.
func WithTLSConfig(cfg *tls.Config) Option {
	return func(c *Client) {
		t, ok := c.t.(*httpTransport)
		if !ok {
			panic("rtorrent: WithTLSConfig used with a non-HTTP transport (Dial/DialUnix); it only applies to DialHTTP")
		}
		transport, ok := t.httpClient.Transport.(*http.Transport)
		if !ok || transport == nil {
			transport = http.DefaultTransport.(*http.Transport).Clone()
		}
		transport.TLSClientConfig = cfg
		t.httpClient.Transport = transport
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

// Close releases resources held by Client. For a Client constructed with
// DialHTTP, it closes idle HTTP connections.
func (c *Client) Close() error {
	if t, ok := c.t.(*httpTransport); ok {
		t.httpClient.CloseIdleConnections()
	}
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

// LoadStart loads a torrent from a filesystem path or URL and starts it
// immediately.
func (c *Client) LoadStart(ctx context.Context, path string) error {
	_, err := c.Call(ctx, "load.start", NewString(""), NewString(path))
	return err
}

// SetCustom1 sets the torrent identified by hash's first custom field.
func (c *Client) SetCustom1(ctx context.Context, hash, value string) error {
	_, err := c.Call(ctx, "d.custom1.set", NewString(hash), NewString(value))
	return err
}

// SetCustom2 sets the torrent identified by hash's second custom field.
func (c *Client) SetCustom2(ctx context.Context, hash, value string) error {
	_, err := c.Call(ctx, "d.custom2.set", NewString(hash), NewString(value))
	return err
}

// SetCustom3 sets the torrent identified by hash's third custom field.
func (c *Client) SetCustom3(ctx context.Context, hash, value string) error {
	_, err := c.Call(ctx, "d.custom3.set", NewString(hash), NewString(value))
	return err
}

// SetCustom4 sets the torrent identified by hash's fourth custom field.
func (c *Client) SetCustom4(ctx context.Context, hash, value string) error {
	_, err := c.Call(ctx, "d.custom4.set", NewString(hash), NewString(value))
	return err
}

// SetCustom5 sets the torrent identified by hash's fifth custom field.
func (c *Client) SetCustom5(ctx context.Context, hash, value string) error {
	_, err := c.Call(ctx, "d.custom5.set", NewString(hash), NewString(value))
	return err
}

// Start starts the torrent identified by hash.
func (c *Client) Start(ctx context.Context, hash string) error {
	_, err := c.Call(ctx, "d.start", NewString(hash))
	return err
}

// Stop stops the torrent identified by hash, announcing "stopped" to its
// trackers and disconnecting its peers.
func (c *Client) Stop(ctx context.Context, hash string) error {
	_, err := c.Call(ctx, "d.stop", NewString(hash))
	return err
}

// Pause pauses the torrent identified by hash without stopping it: peers
// stay connected and no "stopped" event is sent to trackers. Use Stop for
// a full stop.
func (c *Client) Pause(ctx context.Context, hash string) error {
	_, err := c.Call(ctx, "d.pause", NewString(hash))
	return err
}

// Resume resumes a torrent identified by hash previously paused with Pause.
func (c *Client) Resume(ctx context.Context, hash string) error {
	_, err := c.Call(ctx, "d.resume", NewString(hash))
	return err
}

// OpenTorrent opens the torrent identified by hash, allocating its files
// on disk.
func (c *Client) OpenTorrent(ctx context.Context, hash string) error {
	_, err := c.Call(ctx, "d.open", NewString(hash))
	return err
}

// CloseTorrent closes the torrent identified by hash, releasing its file
// handles.
func (c *Client) CloseTorrent(ctx context.Context, hash string) error {
	_, err := c.Call(ctx, "d.close", NewString(hash))
	return err
}

// Erase removes the torrent identified by hash from the session. It
// does not delete the torrent's downloaded data from disk.
func (c *Client) Erase(ctx context.Context, hash string) error {
	_, err := c.Call(ctx, "d.erase", NewString(hash))
	return err
}

// SetPriority sets the torrent identified by hash's priority: 0 (off),
// 1 (low), 2 (normal), or 3 (high).
func (c *Client) SetPriority(ctx context.Context, hash string, priority int64) error {
	_, err := c.Call(ctx, "d.priority.set", NewString(hash), NewInt64(priority))
	return err
}

// SetFilePriority sets the priority of the file at index (0-based, in the
// order Files returns them) within the torrent identified by hash: 0 (off),
// 1 (normal), or 2 (high).
func (c *Client) SetFilePriority(ctx context.Context, hash string, index int, priority int64) error {
	target := fmt.Sprintf("%s:f%d", hash, index)
	_, err := c.Call(ctx, "f.priority.set", NewString(target), NewInt64(priority))
	return err
}

// SetDirectory sets the torrent identified by hash's download directory
// metadata. It does not move the torrent's files on disk.
func (c *Client) SetDirectory(ctx context.Context, hash, path string) error {
	_, err := c.Call(ctx, "d.directory.set", NewString(hash), NewString(path))
	return err
}

// CheckHash triggers a hash check (rehash) of the torrent identified by
// hash.
func (c *Client) CheckHash(ctx context.Context, hash string) error {
	_, err := c.Call(ctx, "d.check_hash", NewString(hash))
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
