package rtorrent

import (
	"context"
	"errors"
	"io"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSCGIRequest(t *testing.T) {
	got := scgiRequest([]byte("hello"))
	want := "24:CONTENT_LENGTH\x005\x00SCGI\x001\x00,hello"
	if string(got) != want {
		t.Errorf("scgiRequest(%q) = %q, want %q", "hello", got, want)
	}
}

func TestSCGIRequestEmptyBody(t *testing.T) {
	got := scgiRequest(nil)
	want := "24:CONTENT_LENGTH\x000\x00SCGI\x001\x00,"
	if string(got) != want {
		t.Errorf("scgiRequest(nil) = %q, want %q", got, want)
	}
}

func TestReadSCGIResponse(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{
			name: "headers and body",
			in:   "Status: 200 OK\r\nContent-Type: text/xml\r\n\r\n<methodResponse/>",
			want: "<methodResponse/>",
		},
		{
			name: "no headers, blank line only",
			in:   "\r\n<methodResponse/>",
			want: "<methodResponse/>",
		},
		{
			name: "empty body",
			in:   "Status: 200 OK\r\n\r\n",
			want: "",
		},
		{
			name:    "truncated, no blank line",
			in:      "Status: 200 OK\r\nContent-Type: text/xml",
			wantErr: true,
		},
		{
			name:    "empty stream",
			in:      "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readSCGIResponse(strings.NewReader(tt.in))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("readSCGIResponse(%q) error = nil, want error", tt.in)
				}
				return
			}
			if err != nil {
				t.Fatalf("readSCGIResponse(%q) unexpected error: %v", tt.in, err)
			}
			if string(got) != tt.want {
				t.Errorf("readSCGIResponse(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func scgiFakeServer(t *testing.T, network, address string, handle func(net.Conn)) string {
	t.Helper()

	ln, err := net.Listen(network, address)
	if err != nil {
		t.Fatalf("net.Listen(%q, %q): %v", network, address, err)
	}
	t.Cleanup(func() { ln.Close() })

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go handle(conn)
		}
	}()

	return ln.Addr().String()
}

func respondSCGI(conn net.Conn, body string) {
	defer conn.Close()
	io.Copy(io.Discard, conn)
	conn.Write([]byte("Status: 200 OK\r\n\r\n" + body))
}

func TestSCGITransportCallSuccessTCP(t *testing.T) {
	addr := scgiFakeServer(t, "tcp", "127.0.0.1:0", func(conn net.Conn) {
		respondSCGI(conn, "<methodResponse/>")
	})

	s := &scgiTransport{network: "tcp", address: addr, timeout: 5 * time.Second}
	got, err := s.call(context.Background(), []byte("request"))
	if err != nil {
		t.Fatalf("call() unexpected error: %v", err)
	}
	if string(got) != "<methodResponse/>" {
		t.Errorf("call() = %q, want %q", got, "<methodResponse/>")
	}
}

func TestSCGITransportCallSuccessUnix(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "rtorrent.sock")
	addr := scgiFakeServer(t, "unix", sock, func(conn net.Conn) {
		respondSCGI(conn, "<methodResponse/>")
	})

	s := &scgiTransport{network: "unix", address: addr, timeout: 5 * time.Second}
	got, err := s.call(context.Background(), []byte("request"))
	if err != nil {
		t.Fatalf("call() unexpected error: %v", err)
	}
	if string(got) != "<methodResponse/>" {
		t.Errorf("call() = %q, want %q", got, "<methodResponse/>")
	}
}

func TestSCGITransportCallDialCanceled(t *testing.T) {
	s := &scgiTransport{network: "tcp", address: "127.0.0.1:0", timeout: 5 * time.Second}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.call(ctx, []byte("request"))
	if err == nil {
		t.Fatal("call() error = nil, want error for already-canceled context")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("call() error = %v, want it to wrap context.Canceled", err)
	}
}

func TestSCGITransportCallReadCanceled(t *testing.T) {
	unblock := make(chan struct{})
	addr := scgiFakeServer(t, "tcp", "127.0.0.1:0", func(conn net.Conn) {
		defer conn.Close()
		io.Copy(io.Discard, conn)
		<-unblock
	})
	t.Cleanup(func() { close(unblock) })

	s := &scgiTransport{network: "tcp", address: addr}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := s.call(ctx, []byte("request"))
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("call() error = nil, want error from context deadline")
	}
	if elapsed > 2*time.Second {
		t.Errorf("call() took %v, want it to abort promptly on ctx deadline", elapsed)
	}
}

func TestSCGITransportCallTimeoutFallback(t *testing.T) {
	unblock := make(chan struct{})
	addr := scgiFakeServer(t, "tcp", "127.0.0.1:0", func(conn net.Conn) {
		defer conn.Close()
		io.Copy(io.Discard, conn)
		<-unblock
	})
	t.Cleanup(func() { close(unblock) })

	s := &scgiTransport{network: "tcp", address: addr, timeout: 100 * time.Millisecond}

	start := time.Now()
	_, err := s.call(context.Background(), []byte("request"))
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("call() error = nil, want timeout error")
	}
	if elapsed > 2*time.Second {
		t.Errorf("call() took %v, want it to abort promptly on the fallback timeout", elapsed)
	}
}
