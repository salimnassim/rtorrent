package rtorrent

import (
	"bytes"
	"context"
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
