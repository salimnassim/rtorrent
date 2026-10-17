package rtorrent

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestClientTorrentsEndToEnd(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><array><data>` +
		`<value><array><data>` +
		`<value><string>0123456789ABCDEF0123456789ABCDEF01234567</string></value>` + // d.hash=
		`<value><string>ubuntu.iso</string></value>` + // d.name=
		`<value><i8>1000000000</i8></value>` + // d.size_bytes=
		`<value><i8>500000000</i8></value>` + // d.completed_bytes=
		`<value><i8>500000000</i8></value>` + // d.left_bytes=
		`<value><i8>1024</i8></value>` + // d.down.rate=
		`<value><i8>2048</i8></value>` + // d.up.rate=
		`<value><i8>500000000</i8></value>` + // d.down.total=
		`<value><i8>250000000</i8></value>` + // d.up.total=
		`<value><i8>500</i8></value>` + // d.ratio= (x1000)
		`<value><i8>1</i8></value>` + // d.state=
		`<value><i8>1</i8></value>` + // d.is_active=
		`<value><i8>1</i8></value>` + // d.is_open=
		`<value><i8>0</i8></value>` + // d.is_multi_file=
		`<value><i8>0</i8></value>` + // d.is_private=
		`<value><string></string></value>` + // d.message=
		`<value><string>/data/ubuntu.iso</string></value>` + // d.base_path=
		`<value><string>/data</string></value>` + // d.directory=
		`<value><i8>2</i8></value>` + // d.priority=
		`<value><string>movies</string></value>` + // d.custom1=
		`<value><string></string></value>` + // d.custom2=
		`<value><string></string></value>` + // d.custom3=
		`<value><string></string></value>` + // d.custom4=
		`<value><string></string></value>` + // d.custom5=
		`<value><i8>0</i8></value>` + // d.hashing=
		`</data></array></value>` +
		`</data></array></value></param></params></methodResponse>`

	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			return []byte(resp), nil
		},
	}, nil)

	torrents, err := c.Torrents(t.Context(), "main")
	if err != nil {
		t.Fatalf("Torrents() unexpected error: %v", err)
	}
	if len(torrents) != 1 {
		t.Fatalf("Torrents() returned %d torrents, want 1", len(torrents))
	}

	got := torrents[0]
	want := &Torrent{
		Hash:           "0123456789ABCDEF0123456789ABCDEF01234567",
		Name:           "ubuntu.iso",
		SizeBytes:      1000000000,
		CompletedBytes: 500000000,
		LeftBytes:      500000000,
		DownRate:       1024,
		UpRate:         2048,
		DownTotal:      500000000,
		UpTotal:        250000000,
		Ratio:          500,
		State:          1,
		IsActive:       true,
		IsOpen:         true,
		IsMultiFile:    false,
		IsPrivate:      false,
		Message:        "",
		BasePath:       "/data/ubuntu.iso",
		Directory:      "/data",
		Priority:       2,
		Custom1:        "movies",
		Hashing:        0,
	}

	if *got != *want {
		t.Errorf("Torrents()[0] = %+v, want %+v", got, want)
	}
}

func TestTorrentFromRowColumnMismatch(t *testing.T) {
	_, err := torrentFromRow([]Value{NewString("only one column")})
	if err == nil {
		t.Fatal("torrentFromRow() error = nil, want error for column count mismatch")
	}
}

const testHash = "0123456789ABCDEF0123456789ABCDEF01234567"

func multicallResponse(results ...string) []byte {
	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0"?><methodResponse><params><param><value><array><data>`)
	for _, r := range results {
		sb.WriteString(r)
	}
	sb.WriteString(`</data></array></value></param></params></methodResponse>`)
	return []byte(sb.String())
}

// okResult is a successful system.multicall result: a one-element array.
func okResult(inner string) string {
	return `<value><array><data><value>` + inner + `</value></data></array></value>`
}

const infoHashFault = `<value><struct>` +
	`<member><name>faultCode</name><value><i4>-501</i4></value></member>` +
	`<member><name>faultString</name><value><string>Could not find info-hash.</string></value></member>` +
	`</struct></value>`

func TestClientTorrentCustom(t *testing.T) {
	var gotBody []byte
	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			gotBody = body
			return multicallResponse(
				okResult(`<string>ubuntu.iso</string>`),
				okResult(`<i8>1024</i8>`),
			), nil
		},
	}, nil)

	got, err := c.TorrentCustom(t.Context(), testHash, "d.name=", "d.down.rate=")
	if err != nil {
		t.Fatalf("TorrentCustom() unexpected error: %v", err)
	}
	want := []Value{NewString("ubuntu.iso"), NewInt64(1024)}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(Value{})); diff != "" {
		t.Errorf("TorrentCustom() mismatch (-want +got):\n%s", diff)
	}

	for _, name := range []string{"d.name", "d.down.rate"} {
		if !bytes.Contains(gotBody, []byte("<string>"+name+"</string>")) {
			t.Errorf("request body missing methodName %q: %s", name, gotBody)
		}
	}
	if bytes.Contains(gotBody, []byte("d.name=")) {
		t.Errorf("request body contains untrimmed command %q: %s", "d.name=", gotBody)
	}
}

func TestClientTorrentNotFound(t *testing.T) {
	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			faults := make([]string, len(torrentColumns))
			for i := range faults {
				faults[i] = infoHashFault
			}
			return multicallResponse(faults...), nil
		},
	}, nil)

	_, err := c.Torrent(t.Context(), testHash)
	if !errors.Is(err, ErrTorrentNotFound) {
		t.Errorf("Torrent() error = %v, want it to match ErrTorrentNotFound", err)
	}
	var fault *Fault
	if !errors.As(err, &fault) {
		t.Errorf("errors.As(%v, *Fault) = false, want true", err)
	}
}

func TestSystemMulticallRowErrors(t *testing.T) {
	otherFault := `<value><struct>` +
		`<member><name>faultCode</name><value><i4>-500</i4></value></member>` +
		`<member><name>faultString</name><value><string>Method not defined</string></value></member>` +
		`</struct></value>`

	tests := []struct {
		name      string
		resp      []byte
		wantFault bool
	}{
		{name: "other fault is not not-found", resp: multicallResponse(otherFault), wantFault: true},
		{name: "result count mismatch", resp: multicallResponse(okResult(`<i8>1</i8>`), okResult(`<i8>2</i8>`))},
		{name: "result is a scalar", resp: multicallResponse(`<value><i8>1</i8></value>`)},
		{name: "multi-element result", resp: multicallResponse(`<value><array><data><value><i8>1</i8></value><value><i8>2</i8></value></data></array></value>`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newClient(&stubTransport{
				callFunc: func(ctx context.Context, body []byte) ([]byte, error) { return tt.resp, nil },
			}, nil)

			_, err := c.TorrentCustom(t.Context(), testHash, "d.name=")
			if err == nil {
				t.Fatal("TorrentCustom() error = nil, want error")
			}
			if errors.Is(err, ErrTorrentNotFound) {
				t.Errorf("TorrentCustom() error = %v, must not match ErrTorrentNotFound", err)
			}
			var fault *Fault
			if got := errors.As(err, &fault); got != tt.wantFault {
				t.Errorf("errors.As(%v, *Fault) = %v, want %v", err, got, tt.wantFault)
			}
		})
	}
}

func TestClientTorrent(t *testing.T) {
	results := make([]string, len(torrentColumns))
	for i, col := range torrentColumns {
		switch col {
		case "d.hash=":
			results[i] = okResult(`<string>` + testHash + `</string>`)
		case "d.name=", "d.message=", "d.base_path=", "d.directory=",
			"d.custom1=", "d.custom2=", "d.custom3=", "d.custom4=", "d.custom5=":
			results[i] = okResult(`<string>x</string>`)
		default:
			results[i] = okResult(`<i8>1</i8>`)
		}
	}
	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			return multicallResponse(results...), nil
		},
	}, nil)

	got, err := c.Torrent(t.Context(), testHash)
	if err != nil {
		t.Fatalf("Torrent() unexpected error: %v", err)
	}
	if got.Hash != testHash || got.Name != "x" || got.SizeBytes != 1 || !got.IsActive || got.Hashing != 1 {
		t.Errorf("Torrent() = %+v, want fields populated from each column", got)
	}
}
