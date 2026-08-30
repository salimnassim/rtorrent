package rtorrent

import (
	"context"
	"testing"
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

	torrents, err := c.Torrents(context.Background(), "main")
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
