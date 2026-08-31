package rtorrent

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type stubTransport struct {
	callFunc func(ctx context.Context, body []byte) ([]byte, error)
}

func (s *stubTransport) call(ctx context.Context, body []byte) ([]byte, error) {
	return s.callFunc(ctx, body)
}

func TestClientCallDecodesResponse(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><string>ok</string></value></param></params></methodResponse>`

	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			return []byte(resp), nil
		},
	}, nil)

	v, err := c.Call(context.Background(), "system.listMethods")
	if err != nil {
		t.Fatalf("Call() unexpected error: %v", err)
	}
	got, err := v.AsString()
	if err != nil {
		t.Fatalf("AsString() unexpected error: %v", err)
	}
	if got != "ok" {
		t.Errorf("Call() = %q, want %q", got, "ok")
	}
}

func TestClientCallFaultUnwrapsViaErrorsAs(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><fault><value><struct>` +
		`<member><name>faultCode</name><value><int>500</int></value></member>` +
		`<member><name>faultString</name><value><string>method not found</string></value></member>` +
		`</struct></value></fault></methodResponse>`

	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			return []byte(resp), nil
		},
	}, nil)

	_, err := c.Call(context.Background(), "bogus.method")
	if err == nil {
		t.Fatal("Call() error = nil, want fault error")
	}

	var fault *Fault
	if !errors.As(err, &fault) {
		t.Fatalf("errors.As(%v, *Fault) = false, want true", err)
	}
	if fault.FaultCode != 500 || fault.FaultString != "method not found" {
		t.Errorf("fault = %+v, want {500 method not found}", fault)
	}
}

func TestClientCallTimeoutPrecedence(t *testing.T) {
	tests := []struct {
		name        string
		ctxDeadline time.Duration
		clientOpts  []Option
		wantElapsed time.Duration
	}{
		{
			name:        "ctx deadline wins over configured timeout",
			ctxDeadline: 50 * time.Millisecond,
			clientOpts:  []Option{WithTimeout(time.Hour)},
			wantElapsed: 50 * time.Millisecond,
		},
		{
			name:        "configured timeout used when ctx has no deadline",
			clientOpts:  []Option{WithTimeout(50 * time.Millisecond)},
			wantElapsed: 50 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			blocked := make(chan struct{})
			c := newClient(&stubTransport{
				callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
					<-ctx.Done()
					close(blocked)
					return nil, ctx.Err()
				},
			}, tt.clientOpts)

			ctx := context.Background()
			var cancel context.CancelFunc
			if tt.ctxDeadline > 0 {
				ctx, cancel = context.WithTimeout(ctx, tt.ctxDeadline)
				defer cancel()
			}

			start := time.Now()
			_, err := c.Call(ctx, "system.listMethods")
			elapsed := time.Since(start)

			<-blocked

			if err == nil {
				t.Fatal("Call() error = nil, want timeout error")
			}
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("Call() error = %v, want it to wrap context.DeadlineExceeded", err)
			}
			if elapsed > 2*time.Second {
				t.Errorf("Call() took %v, want it to abort near %v", elapsed, tt.wantElapsed)
			}
		})
	}
}

func TestClientLoadRawStart(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><i4>0</i4></value></param></params></methodResponse>`

	var gotBody []byte
	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			gotBody = body
			return []byte(resp), nil
		},
	}, nil)

	if err := c.LoadRawStart(context.Background(), []byte("d4:name4:fakee")); err != nil {
		t.Fatalf("LoadRawStart() unexpected error: %v", err)
	}
	if !bytes.Contains(gotBody, []byte("<methodName>load.raw_start</methodName>")) {
		t.Errorf("LoadRawStart() body = %s, want load.raw_start method call", gotBody)
	}
	if !bytes.Contains(gotBody, []byte("<base64>ZDQ6bmFtZTQ6ZmFrZWU=</base64>")) {
		t.Errorf("LoadRawStart() body = %s, want base64-encoded data", gotBody)
	}
}

func TestClientLoadStart(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><i4>0</i4></value></param></params></methodResponse>`

	var gotBody []byte
	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			gotBody = body
			return []byte(resp), nil
		},
	}, nil)

	if err := c.LoadStart(context.Background(), "/downloads/fake.torrent"); err != nil {
		t.Fatalf("LoadStart() unexpected error: %v", err)
	}
	if !bytes.Contains(gotBody, []byte("<methodName>load.start</methodName>")) {
		t.Errorf("LoadStart() body = %s, want load.start method call", gotBody)
	}
	if !bytes.Contains(gotBody, []byte("<string>/downloads/fake.torrent</string>")) {
		t.Errorf("LoadStart() body = %s, want path string", gotBody)
	}
}

func TestClientSetCustom(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><i4>0</i4></value></param></params></methodResponse>`

	tests := []struct {
		name       string
		setCustom  func(c *Client, ctx context.Context, hash, value string) error
		wantMethod string
	}{
		{name: "SetCustom1", setCustom: (*Client).SetCustom1, wantMethod: "d.custom1.set"},
		{name: "SetCustom2", setCustom: (*Client).SetCustom2, wantMethod: "d.custom2.set"},
		{name: "SetCustom3", setCustom: (*Client).SetCustom3, wantMethod: "d.custom3.set"},
		{name: "SetCustom4", setCustom: (*Client).SetCustom4, wantMethod: "d.custom4.set"},
		{name: "SetCustom5", setCustom: (*Client).SetCustom5, wantMethod: "d.custom5.set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody []byte
			c := newClient(&stubTransport{
				callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
					gotBody = body
					return []byte(resp), nil
				},
			}, nil)

			if err := tt.setCustom(c, context.Background(), "0123456789ABCDEF0123456789ABCDEF01234567", "movies"); err != nil {
				t.Fatalf("%s() unexpected error: %v", tt.name, err)
			}
			if !bytes.Contains(gotBody, []byte("<methodName>"+tt.wantMethod+"</methodName>")) {
				t.Errorf("%s() body = %s, want %s method call", tt.name, gotBody, tt.wantMethod)
			}
			if !bytes.Contains(gotBody, []byte("<string>movies</string>")) {
				t.Errorf("%s() body = %s, want value string", tt.name, gotBody)
			}
		})
	}
}

func TestClientTorrentAction(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><i4>0</i4></value></param></params></methodResponse>`

	tests := []struct {
		name       string
		action     func(c *Client, ctx context.Context, hash string) error
		wantMethod string
	}{
		{name: "Start", action: (*Client).Start, wantMethod: "d.start"},
		{name: "Stop", action: (*Client).Stop, wantMethod: "d.stop"},
		{name: "Pause", action: (*Client).Pause, wantMethod: "d.pause"},
		{name: "Resume", action: (*Client).Resume, wantMethod: "d.resume"},
		{name: "OpenTorrent", action: (*Client).OpenTorrent, wantMethod: "d.open"},
		{name: "CloseTorrent", action: (*Client).CloseTorrent, wantMethod: "d.close"},
		{name: "Erase", action: (*Client).Erase, wantMethod: "d.erase"},
		{name: "CheckHash", action: (*Client).CheckHash, wantMethod: "d.check_hash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotBody []byte
			c := newClient(&stubTransport{
				callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
					gotBody = body
					return []byte(resp), nil
				},
			}, nil)

			const hash = "0123456789ABCDEF0123456789ABCDEF01234567"
			if err := tt.action(c, context.Background(), hash); err != nil {
				t.Fatalf("%s() unexpected error: %v", tt.name, err)
			}
			if !bytes.Contains(gotBody, []byte("<methodName>"+tt.wantMethod+"</methodName>")) {
				t.Errorf("%s() body = %s, want %s method call", tt.name, gotBody, tt.wantMethod)
			}
			if !bytes.Contains(gotBody, []byte("<string>"+hash+"</string>")) {
				t.Errorf("%s() body = %s, want hash string", tt.name, gotBody)
			}
		})
	}
}

func TestClientSetPriority(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><i4>0</i4></value></param></params></methodResponse>`

	var gotBody []byte
	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			gotBody = body
			return []byte(resp), nil
		},
	}, nil)

	if err := c.SetPriority(context.Background(), "0123456789ABCDEF0123456789ABCDEF01234567", 3); err != nil {
		t.Fatalf("SetPriority() unexpected error: %v", err)
	}
	if !bytes.Contains(gotBody, []byte("<methodName>d.priority.set</methodName>")) {
		t.Errorf("SetPriority() body = %s, want d.priority.set method call", gotBody)
	}
	if !bytes.Contains(gotBody, []byte("<string>0123456789ABCDEF0123456789ABCDEF01234567</string>")) {
		t.Errorf("SetPriority() body = %s, want hash string", gotBody)
	}
	if !bytes.Contains(gotBody, []byte("<i8>3</i8>")) {
		t.Errorf("SetPriority() body = %s, want priority value 3", gotBody)
	}
}

func TestClientSetFilePriority(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><i4>0</i4></value></param></params></methodResponse>`

	var gotBody []byte
	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			gotBody = body
			return []byte(resp), nil
		},
	}, nil)

	if err := c.SetFilePriority(context.Background(), "0123456789ABCDEF0123456789ABCDEF01234567", 2, 1); err != nil {
		t.Fatalf("SetFilePriority() unexpected error: %v", err)
	}
	if !bytes.Contains(gotBody, []byte("<methodName>f.priority.set</methodName>")) {
		t.Errorf("SetFilePriority() body = %s, want f.priority.set method call", gotBody)
	}
	if !bytes.Contains(gotBody, []byte("<string>0123456789ABCDEF0123456789ABCDEF01234567:f2</string>")) {
		t.Errorf("SetFilePriority() body = %s, want compound file target", gotBody)
	}
	if !bytes.Contains(gotBody, []byte("<i8>1</i8>")) {
		t.Errorf("SetFilePriority() body = %s, want priority value 1", gotBody)
	}
}

func TestClientSetDirectory(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><i4>0</i4></value></param></params></methodResponse>`

	var gotBody []byte
	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			gotBody = body
			return []byte(resp), nil
		},
	}, nil)

	if err := c.SetDirectory(context.Background(), "0123456789ABCDEF0123456789ABCDEF01234567", "/downloads/moved"); err != nil {
		t.Fatalf("SetDirectory() unexpected error: %v", err)
	}
	if !bytes.Contains(gotBody, []byte("<methodName>d.directory.set</methodName>")) {
		t.Errorf("SetDirectory() body = %s, want d.directory.set method call", gotBody)
	}
	if !bytes.Contains(gotBody, []byte("<string>/downloads/moved</string>")) {
		t.Errorf("SetDirectory() body = %s, want path string", gotBody)
	}
}

func TestClientDialHTTPWithBasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "user" || pass != "pass" {
			t.Errorf("BasicAuth() = (%q, %q, %v), want (%q, %q, true)", user, pass, ok, "user", "pass")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<?xml version="1.0"?><methodResponse><params><param><value><string>ok</string></value></param></params></methodResponse>`))
	}))
	t.Cleanup(srv.Close)

	c := DialHTTP(srv.URL, WithBasicAuth("user", "pass"))
	if _, err := c.Call(context.Background(), "system.listMethods"); err != nil {
		t.Fatalf("Call() unexpected error: %v", err)
	}
}

func TestWithBasicAuthPanicsOnNonHTTPTransport(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Dial(WithBasicAuth) did not panic, want panic")
		}
	}()
	Dial("127.0.0.1:5000", WithBasicAuth("user", "pass"))
}

func TestClientDialHTTPWithTLSConfig(t *testing.T) {
	cfg := &tls.Config{InsecureSkipVerify: true}
	c := DialHTTP("http://127.0.0.1:5000", WithTLSConfig(cfg))

	ht := c.t.(*httpTransport)
	transport, ok := ht.httpClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("httpClient.Transport = %T, want *http.Transport", ht.httpClient.Transport)
	}
	if transport.TLSClientConfig != cfg {
		t.Errorf("TLSClientConfig = %p, want %p", transport.TLSClientConfig, cfg)
	}
}

func TestWithTLSConfigPanicsOnNonHTTPTransport(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Dial(WithTLSConfig) did not panic, want panic")
		}
	}()
	Dial("127.0.0.1:5000", WithTLSConfig(&tls.Config{}))
}

func TestClientMulticallRows(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><array><data>` +
		`<value><array><data><value><string>hash1</string></value><value><string>name1</string></value></data></array></value>` +
		`<value><array><data><value><string>hash2</string></value><value><string>name2</string></value></data></array></value>` +
		`</data></array></value></param></params></methodResponse>`

	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			return []byte(resp), nil
		},
	}, nil)

	rows, err := c.Multicall(context.Background(), "d.multicall2", []Value{NewString(""), NewString("main")}, "d.hash=", "d.name=")
	if err != nil {
		t.Fatalf("Multicall() unexpected error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("Multicall() returned %d rows, want 2", len(rows))
	}
	hash, err := rows[0][0].AsString()
	if err != nil {
		t.Fatalf("AsString() unexpected error: %v", err)
	}
	if hash != "hash1" {
		t.Errorf("rows[0][0] = %q, want %q", hash, "hash1")
	}
}

func TestClientMulticallRowLengthMismatch(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><array><data>` +
		`<value><array><data><value><string>hash1</string></value></data></array></value>` +
		`</data></array></value></param></params></methodResponse>`

	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			return []byte(resp), nil
		},
	}, nil)

	_, err := c.Multicall(context.Background(), "d.multicall2", []Value{NewString(""), NewString("main")}, "d.hash=", "d.name=")
	if err == nil {
		t.Fatal("Multicall() error = nil, want row-length mismatch error")
	}
}

func TestClientMulticallNotAnArray(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><string>not an array</string></value></param></params></methodResponse>`

	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			return []byte(resp), nil
		},
	}, nil)

	_, err := c.Multicall(context.Background(), "d.multicall2", []Value{NewString(""), NewString("main")}, "d.hash=")
	if err == nil {
		t.Fatal("Multicall() error = nil, want error for non-array response")
	}
}

func TestClientClose(t *testing.T) {
	if err := DialHTTP("https://example.com/RPC2").Close(); err != nil {
		t.Errorf("Close() on an HTTP client = %v, want nil", err)
	}
	if err := Dial("127.0.0.1:5000").Close(); err != nil {
		t.Errorf("Close() on an SCGI client = %v, want nil", err)
	}
}
