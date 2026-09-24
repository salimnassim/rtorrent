package main

import (
	"io"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/salimnassim/rtorrent"
)

func TestDial(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{name: "bare tcp address", addr: "127.0.0.1:5000"},
		{name: "scgi scheme", addr: "scgi://127.0.0.1:5000"},
		{name: "unix scheme", addr: "unix:///tmp/rtorrent.sock"},
		{name: "http scheme", addr: "http://example.com/RPC2"},
		{name: "https scheme", addr: "https://example.com/RPC2"},
		{name: "unsupported scheme", addr: "ftp://example.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := dial(tt.addr, "", "", time.Second)
			if tt.wantErr {
				if err == nil {
					t.Errorf("dial(%q) error = nil, want error", tt.addr)
				}
				return
			}
			if err != nil {
				t.Fatalf("dial(%q) unexpected error: %v", tt.addr, err)
			}
			if client == nil {
				t.Errorf("dial(%q) client = nil, want non-nil", tt.addr)
			}
		})
	}
}

func TestToAny(t *testing.T) {
	tests := []struct {
		name  string
		value rtorrent.Value
		want  any
	}{
		{name: "string", value: rtorrent.NewString("foo"), want: "foo"},
		{name: "int", value: rtorrent.NewInt(5), want: int64(5)},
		{name: "int64", value: rtorrent.NewInt64(5), want: int64(5)},
		{name: "double", value: rtorrent.NewDouble(1.5), want: 1.5},
		{name: "bool", value: rtorrent.NewBool(true), want: true},
		{name: "nil", value: rtorrent.NewNil(), want: nil},
		{name: "base64", value: rtorrent.NewBase64([]byte{0, 1, 2}), want: []byte{0, 1, 2}},
		{
			name:  "array",
			value: rtorrent.NewArray([]rtorrent.Value{rtorrent.NewString("a"), rtorrent.NewInt(1)}),
			want:  []any{"a", int64(1)},
		},
		{
			name:  "struct",
			value: rtorrent.NewStruct(map[string]rtorrent.Value{"k": rtorrent.NewString("v")}),
			want:  map[string]any{"k": "v"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toAny(tt.value)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("toAny(%v) mismatch (-want +got):\n%s", tt.value, diff)
			}
		})
	}
}

func TestRunEnvFallback(t *testing.T) {
	env := map[string]string{"RTCTL_ADDR": "ftp://example.com"}
	getenv := func(key string) string { return env[key] }

	err := run(t.Context(), []string{"system.listMethods"}, getenv, io.Discard)
	if err == nil || !strings.Contains(err.Error(), `unsupported address scheme "ftp"`) {
		t.Errorf("run() error = %v, want unsupported scheme error from RTCTL_ADDR", err)
	}

	// The flag must win over the env: a gopher:// -addr fails before any dial.
	err = run(t.Context(), []string{"-addr", "gopher://example.com", "system.listMethods"}, getenv, io.Discard)
	if err == nil || !strings.Contains(err.Error(), `unsupported address scheme "gopher"`) {
		t.Errorf("run(-addr gopher://...) error = %v, want flag to override RTCTL_ADDR", err)
	}

	err = run(t.Context(), []string{"system.listMethods"}, func(string) string { return "" }, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "-addr is required") {
		t.Errorf("run() with no addr error = %v, want -addr is required", err)
	}
}
