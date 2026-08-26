package rtorrent

import (
	"context"
	"fmt"
)

// trackerColumns is the fixed list of t.* commands passed to t.multicall to
// populate a Tracker, in the order Tracker's fields are read by
// trackerFromRow.
var trackerColumns = []string{
	"t.url=",
	"t.type=",
	"t.group=",
	"t.id=",
	"t.is_enabled=",
	"t.is_usable=",
	"t.scrape_complete=",
	"t.scrape_incomplete=",
}

// Tracker is a snapshot of one tracker attached to a torrent.
type Tracker struct {
	// URL is the tracker's announce URL.
	URL string
	// Type is the tracker protocol: 1 (HTTP), 2 (UDP), or 3 (DHT).
	Type int64
	// Group is the tracker's group number (trackers in the same group are
	// tried together).
	Group int64
	// ID is the tracker_id the tracker returned in a previous announce, or
	// empty if none.
	ID string
	// IsEnabled reports whether the tracker is enabled.
	IsEnabled bool
	// IsUsable reports whether the tracker is currently usable (enabled and
	// not in an error backoff state).
	IsUsable bool
	// ScrapeComplete is the seeder count from the tracker's last scrape.
	ScrapeComplete int64
	// ScrapeIncomplete is the leecher count from the tracker's last scrape.
	ScrapeIncomplete int64
}

// trackerFromRow converts one t.multicall result row into a Tracker.
func trackerFromRow(row []Value) (*Tracker, error) {
	if len(row) != len(trackerColumns) {
		return nil, fmt.Errorf("rtorrent: tracker row: got %d columns, want %d", len(row), len(trackerColumns))
	}

	var (
		t   Tracker
		err error
	)
	if t.URL, err = row[0].AsString(); err != nil {
		return nil, fmt.Errorf("rtorrent: tracker row: url: %w", err)
	}
	if t.Type, err = row[1].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: tracker row: type: %w", err)
	}
	if t.Group, err = row[2].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: tracker row: group: %w", err)
	}
	if t.ID, err = row[3].AsString(); err != nil {
		return nil, fmt.Errorf("rtorrent: tracker row: id: %w", err)
	}

	isEnabled, err := row[4].AsInt64()
	if err != nil {
		return nil, fmt.Errorf("rtorrent: tracker row: is enabled: %w", err)
	}
	t.IsEnabled = isEnabled != 0

	isUsable, err := row[5].AsInt64()
	if err != nil {
		return nil, fmt.Errorf("rtorrent: tracker row: is usable: %w", err)
	}
	t.IsUsable = isUsable != 0

	if t.ScrapeComplete, err = row[6].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: tracker row: scrape complete: %w", err)
	}
	if t.ScrapeIncomplete, err = row[7].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: tracker row: scrape incomplete: %w", err)
	}

	return &t, nil
}

// Trackers returns the trackers attached to the torrent identified by hash.
func (c *Client) Trackers(ctx context.Context, hash string) ([]*Tracker, error) {
	rows, err := c.TrackersCustom(ctx, hash, trackerColumns...)
	if err != nil {
		return nil, err
	}

	trackers := make([]*Tracker, len(rows))
	for i, row := range rows {
		t, err := trackerFromRow(row)
		if err != nil {
			return nil, err
		}
		trackers[i] = t
	}
	return trackers, nil
}

// TrackersCustom calls t.multicall against the torrent identified by hash
// with cmds, returning one raw row of Values per tracker.
func (c *Client) TrackersCustom(ctx context.Context, hash string, cmds ...string) ([][]Value, error) {
	leading := []Value{NewString(hash), NewString("")}
	return c.Multicall(ctx, "t.multicall", leading, cmds...)
}
