package main

import (
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
