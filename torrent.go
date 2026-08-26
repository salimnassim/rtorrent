package rtorrent

import (
	"context"
	"fmt"
)

// torrentColumns is the fixed list of d.* commands passed to d.multicall to
// populate a Torrent, in the order Torrent's fields are read by
// torrentFromRow.
//
// Column order here must exactly match the read order in
// torrentFromRow; this is enforced only by a length check at runtime.
var torrentColumns = []string{
	"d.hash=",
	"d.name=",
	"d.size_bytes=",
	"d.completed_bytes=",
	"d.left_bytes=",
	"d.down.rate=",
	"d.up.rate=",
	"d.down.total=",
	"d.up.total=",
	"d.ratio=",
	"d.state=",
	"d.is_active=",
	"d.is_open=",
	"d.is_multi_file=",
	"d.is_private=",
	"d.message=",
	"d.base_path=",
	"d.directory=",
	"d.priority=",
}

// Torrent is a snapshot of a download.
type Torrent struct {
	// Hash is the torrent's info hash, as 40 uppercase hex characters.
	Hash string
	// Name is the torrent's display name.
	Name string
	// SizeBytes is the total size of the torrent's data.
	SizeBytes int64
	// CompletedBytes is the number of bytes downloaded and verified so far.
	CompletedBytes int64
	// LeftBytes is the number of bytes remaining to download.
	LeftBytes int64
	// DownRate is the current download rate, in bytes per second.
	DownRate int64
	// UpRate is the current upload rate, in bytes per second.
	UpRate int64
	// DownTotal is the total bytes downloaded since the torrent was loaded.
	DownTotal int64
	// UpTotal is the total bytes uploaded since the torrent was loaded.
	UpTotal int64
	// Ratio is the upload ratio, multiplied by 1000.
	Ratio int64
	// State is 0 if the torrent is stopped, 1 if started.
	State int64
	// IsActive reports whether the torrent is currently active.
	IsActive bool
	// IsOpen reports whether the torrent's files are open.
	IsOpen bool
	// IsMultiFile reports whether the torrent contains more than one file.
	IsMultiFile bool
	// IsPrivate reports whether the torrent is private (no DHT/PEX).
	IsPrivate bool
	// Message is the last error or status message rTorrent recorded for
	// this torrent, or empty if none.
	Message string
	// BasePath is the torrent's base path on disk.
	BasePath string
	// Directory is the torrent's download directory.
	Directory string
	// Priority is 0 (off), 1 (low), 2 (normal), or 3 (high).
	Priority int64
}

// torrentFromRow converts one d.multicall result row into a Torrent. Column
// order must match torrentColumns.
func torrentFromRow(row []Value) (*Torrent, error) {
	if len(row) != len(torrentColumns) {
		return nil, fmt.Errorf("rtorrent: torrent row: got %d columns, want %d", len(row), len(torrentColumns))
	}

	var (
		t   Torrent
		err error
	)
	if t.Hash, err = row[0].AsString(); err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: hash: %w", err)
	}
	if t.Name, err = row[1].AsString(); err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: name: %w", err)
	}
	if t.SizeBytes, err = row[2].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: size bytes: %w", err)
	}
	if t.CompletedBytes, err = row[3].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: completed bytes: %w", err)
	}
	if t.LeftBytes, err = row[4].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: left bytes: %w", err)
	}
	if t.DownRate, err = row[5].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: down rate: %w", err)
	}
	if t.UpRate, err = row[6].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: up rate: %w", err)
	}
	if t.DownTotal, err = row[7].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: down total: %w", err)
	}
	if t.UpTotal, err = row[8].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: up total: %w", err)
	}
	if t.Ratio, err = row[9].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: ratio: %w", err)
	}
	if t.State, err = row[10].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: state: %w", err)
	}

	isActive, err := row[11].AsInt64()
	if err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: is active: %w", err)
	}
	t.IsActive = isActive != 0

	isOpen, err := row[12].AsInt64()
	if err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: is open: %w", err)
	}
	t.IsOpen = isOpen != 0

	isMultiFile, err := row[13].AsInt64()
	if err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: is multi file: %w", err)
	}
	t.IsMultiFile = isMultiFile != 0

	isPrivate, err := row[14].AsInt64()
	if err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: is private: %w", err)
	}
	t.IsPrivate = isPrivate != 0

	if t.Message, err = row[15].AsString(); err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: message: %w", err)
	}
	if t.BasePath, err = row[16].AsString(); err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: base path: %w", err)
	}
	if t.Directory, err = row[17].AsString(); err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: directory: %w", err)
	}
	if t.Priority, err = row[18].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: torrent row: priority: %w", err)
	}

	return &t, nil
}

// Torrents returns the torrents visible in view, populated via d.multicall.
// An empty view means the "default" view.
func (c *Client) Torrents(ctx context.Context, view string) ([]*Torrent, error) {
	rows, err := c.TorrentsCustom(ctx, view, torrentColumns...)
	if err != nil {
		return nil, err
	}

	torrents := make([]*Torrent, len(rows))
	for i, row := range rows {
		t, err := torrentFromRow(row)
		if err != nil {
			return nil, err
		}
		torrents[i] = t
	}
	return torrents, nil
}

// TorrentsCustom calls d.multicall against view with cmds, returning one raw
// row of Values per torrent.
func (c *Client) TorrentsCustom(ctx context.Context, view string, cmds ...string) ([][]Value, error) {
	leading := []Value{NewString(""), NewString(view)}
	return c.Multicall(ctx, "d.multicall", leading, cmds...)
}
