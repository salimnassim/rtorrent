package rtorrent

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestClientExecuteThrow(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><i4>0</i4></value></param></params></methodResponse>`

	var gotBody []byte
	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			gotBody = body
			return []byte(resp), nil
		},
	}, nil)

	if err := c.ExecuteThrow(context.Background(), "sh", "-c", "true"); err != nil {
		t.Fatalf("ExecuteThrow() unexpected error: %v", err)
	}
	if !bytes.Contains(gotBody, []byte("<methodName>execute.throw</methodName>")) {
		t.Errorf("ExecuteThrow() body = %s, want execute.throw method call", gotBody)
	}

	wantParams := "<param><value><string></string></value></param>" +
		"<param><value><string>sh</string></value></param>" +
		"<param><value><string>-c</string></value></param>" +
		"<param><value><string>true</string></value></param>"
	if !bytes.Contains(gotBody, []byte(wantParams)) {
		t.Errorf("ExecuteThrow() body = %s, want params %s", gotBody, wantParams)
	}
}

func TestClientExecuteNoArgs(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><i4>0</i4></value></param></params></methodResponse>`

	var gotBody []byte
	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			gotBody = body
			return []byte(resp), nil
		},
	}, nil)

	if err := c.ExecuteThrow(context.Background(), "true"); err != nil {
		t.Fatalf("ExecuteThrow() unexpected error: %v", err)
	}

	wantParams := "<params><param><value><string></string></value></param>" +
		"<param><value><string>true</string></value></param></params>"
	if !bytes.Contains(gotBody, []byte(wantParams)) {
		t.Errorf("ExecuteThrow() body = %s, want params %s", gotBody, wantParams)
	}
}

func TestClientExecuteThrowFaultUnwraps(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><fault><value><struct>` +
		`<member><name>faultCode</name><value><int>1</int></value></member>` +
		`<member><name>faultString</name><value><string>command failed</string></value></member>` +
		`</struct></value></fault></methodResponse>`

	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			return []byte(resp), nil
		},
	}, nil)

	err := c.ExecuteThrow(context.Background(), "false")
	if err == nil {
		t.Fatal("ExecuteThrow() error = nil, want fault error")
	}

	var fault *Fault
	if !errors.As(err, &fault) {
		t.Fatalf("errors.As(%v, *Fault) = false, want true", err)
	}
	if fault.FaultString != "command failed" {
		t.Errorf("fault.FaultString = %q, want %q", fault.FaultString, "command failed")
	}
}

func TestClientExecuteNothrow(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><i4>1</i4></value></param></params></methodResponse>`

	var gotBody []byte
	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			gotBody = body
			return []byte(resp), nil
		},
	}, nil)

	status, err := c.ExecuteNothrow(context.Background(), "sh", "-c", "exit 1")
	if err != nil {
		t.Fatalf("ExecuteNothrow() unexpected error: %v", err)
	}
	if status != 1 {
		t.Errorf("ExecuteNothrow() = %d, want 1", status)
	}
	if !bytes.Contains(gotBody, []byte("<methodName>execute.nothrow</methodName>")) {
		t.Errorf("ExecuteNothrow() body = %s, want execute.nothrow method call", gotBody)
	}
}

func TestClientExecuteNothrowDecodeError(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><string>not an int</string></value></param></params></methodResponse>`

	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			return []byte(resp), nil
		},
	}, nil)

	if _, err := c.ExecuteNothrow(context.Background(), "true"); err == nil {
		t.Fatal("ExecuteNothrow() error = nil, want decode error")
	}
}

func TestClientExecuteCapture(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><string>hello</string></value></param></params></methodResponse>`

	var gotBody []byte
	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			gotBody = body
			return []byte(resp), nil
		},
	}, nil)

	out, err := c.ExecuteCapture(context.Background(), "echo", "-n", "hello")
	if err != nil {
		t.Fatalf("ExecuteCapture() unexpected error: %v", err)
	}
	if out != "hello" {
		t.Errorf("ExecuteCapture() = %q, want %q", out, "hello")
	}
	if !bytes.Contains(gotBody, []byte("<methodName>execute.capture</methodName>")) {
		t.Errorf("ExecuteCapture() body = %s, want execute.capture method call", gotBody)
	}
}

func TestClientExecuteCaptureDecodeError(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><i4>0</i4></value></param></params></methodResponse>`

	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			return []byte(resp), nil
		},
	}, nil)

	if _, err := c.ExecuteCapture(context.Background(), "true"); err == nil {
		t.Fatal("ExecuteCapture() error = nil, want decode error")
	}
}
