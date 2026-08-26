package rtorrent

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPTransportCallSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "text/xml" {
			t.Errorf("Content-Type = %q, want %q", got, "text/xml")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<methodResponse/>"))
	}))
	t.Cleanup(srv.Close)

	tr := &httpTransport{url: srv.URL, httpClient: srv.Client()}
	got, err := tr.call(context.Background(), []byte("request"))
	if err != nil {
		t.Fatalf("call() unexpected error: %v", err)
	}
	if string(got) != "<methodResponse/>" {
		t.Errorf("call() = %q, want %q", got, "<methodResponse/>")
	}
}

func TestHTTPTransportCallBasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("BasicAuth() ok = false, want true")
		}
		if user != "user" || pass != "pass" {
			t.Errorf("BasicAuth() = (%q, %q), want (%q, %q)", user, pass, "user", "pass")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<methodResponse/>"))
	}))
	t.Cleanup(srv.Close)

	tr := &httpTransport{url: srv.URL, httpClient: srv.Client(), username: "user", password: "pass"}
	if _, err := tr.call(context.Background(), []byte("request")); err != nil {
		t.Fatalf("call() unexpected error: %v", err)
	}
}

func TestHTTPTransportCallNoBasicAuth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, _, ok := r.BasicAuth(); ok {
			t.Error("BasicAuth() ok = true, want false when username is unset")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<methodResponse/>"))
	}))
	t.Cleanup(srv.Close)

	tr := &httpTransport{url: srv.URL, httpClient: srv.Client()}
	if _, err := tr.call(context.Background(), []byte("request")); err != nil {
		t.Fatalf("call() unexpected error: %v", err)
	}
}

func TestHTTPTransportCallNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("boom"))
	}))
	t.Cleanup(srv.Close)

	tr := &httpTransport{url: srv.URL, httpClient: srv.Client()}
	_, err := tr.call(context.Background(), []byte("request"))
	if err == nil {
		t.Fatal("call() error = nil, want error for 500 status")
	}
	if !strings.Contains(err.Error(), "500") || !strings.Contains(err.Error(), "boom") {
		t.Errorf("call() error = %v, want it to mention status 500 and body", err)
	}
}

func TestHTTPTransportCallOversizedBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write(make([]byte, maxHTTPResponseBytes+1))
	}))
	t.Cleanup(srv.Close)

	tr := &httpTransport{url: srv.URL, httpClient: srv.Client()}
	_, err := tr.call(context.Background(), []byte("request"))
	if err == nil {
		t.Fatal("call() error = nil, want error for oversized response body")
	}
}

func TestHTTPTransportCallContextCanceled(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)

	tr := &httpTransport{url: srv.URL, httpClient: srv.Client()}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := tr.call(ctx, []byte("request"))
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("call() error = nil, want context deadline error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("call() error = %v, want it to wrap context.DeadlineExceeded", err)
	}
	if elapsed > 2*time.Second {
		t.Errorf("call() took %v, want it to abort promptly on ctx deadline", elapsed)
	}
}
