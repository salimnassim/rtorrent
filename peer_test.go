package rtorrent

import (
	"context"
	"testing"
)

func TestClientPeersEndToEnd(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><array><data>` +
		`<value><array><data>` +
		`<value><string>2d5254313333302d0102030405060708090a</string></value>` + // p.id=
		`<value><string>192.0.2.1</string></value>` + // p.address=
		`<value><i8>51413</i8></value>` + // p.port=
		`<value><string>rTorrent 0.9.8</string></value>` + // p.client_version=
		`<value><i8>1</i8></value>` + // p.is_encrypted=
		`<value><i8>0</i8></value>` + // p.is_incoming=
		`<value><i8>1024</i8></value>` + // p.up_rate=
		`<value><i8>2048</i8></value>` + // p.up_total=
		`<value><i8>4096</i8></value>` + // p.down_rate=
		`<value><i8>8192</i8></value>` + // p.down_total=
		`<value><i8>75</i8></value>` + // p.completed_percent=
		`</data></array></value>` +
		`</data></array></value></param></params></methodResponse>`

	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			return []byte(resp), nil
		},
	}, nil)

	peers, err := c.Peers(context.Background(), "0123456789ABCDEF0123456789ABCDEF01234567")
	if err != nil {
		t.Fatalf("Peers() unexpected error: %v", err)
	}
	if len(peers) != 1 {
		t.Fatalf("Peers() returned %d peers, want 1", len(peers))
	}

	got := peers[0]
	want := &Peer{
		ID:               "2d5254313333302d0102030405060708090a",
		Address:          "192.0.2.1",
		Port:             51413,
		ClientVersion:    "rTorrent 0.9.8",
		IsEncrypted:      true,
		IsIncoming:       false,
		UpRate:           1024,
		UpTotal:          2048,
		DownRate:         4096,
		DownTotal:        8192,
		CompletedPercent: 75,
	}

	if *got != *want {
		t.Errorf("Peers()[0] = %+v, want %+v", got, want)
	}
}

func TestPeerFromRowColumnMismatch(t *testing.T) {
	_, err := peerFromRow([]Value{NewString("only one column")})
	if err == nil {
		t.Fatal("peerFromRow() error = nil, want error for column count mismatch")
	}
}
