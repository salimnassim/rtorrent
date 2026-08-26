package rtorrent

import (
	"context"
	"testing"
)

func TestClientTrackersEndToEnd(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><array><data>` +
		`<value><array><data>` +
		`<value><string>https://tracker.example.com/announce</string></value>` + // t.url=
		`<value><i8>1</i8></value>` + // t.type= (http)
		`<value><i8>0</i8></value>` + // t.group=
		`<value><string></string></value>` + // t.id=
		`<value><i8>1</i8></value>` + // t.is_enabled=
		`<value><i8>1</i8></value>` + // t.is_usable=
		`<value><i8>12</i8></value>` + // t.scrape_complete=
		`<value><i8>3</i8></value>` + // t.scrape_incomplete=
		`</data></array></value>` +
		`</data></array></value></param></params></methodResponse>`

	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			return []byte(resp), nil
		},
	}, nil)

	trackers, err := c.Trackers(context.Background(), "0123456789ABCDEF0123456789ABCDEF01234567")
	if err != nil {
		t.Fatalf("Trackers() unexpected error: %v", err)
	}
	if len(trackers) != 1 {
		t.Fatalf("Trackers() returned %d trackers, want 1", len(trackers))
	}

	got := trackers[0]
	want := &Tracker{
		URL:              "https://tracker.example.com/announce",
		Type:             1,
		Group:            0,
		ID:               "",
		IsEnabled:        true,
		IsUsable:         true,
		ScrapeComplete:   12,
		ScrapeIncomplete: 3,
	}

	if *got != *want {
		t.Errorf("Trackers()[0] = %+v, want %+v", got, want)
	}
}

func TestTrackerFromRowColumnMismatch(t *testing.T) {
	_, err := trackerFromRow([]Value{NewString("only one column")})
	if err == nil {
		t.Fatal("trackerFromRow() error = nil, want error for column count mismatch")
	}
}
