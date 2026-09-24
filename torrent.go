package rtorrent

import "context"

// torrentColumns is the fixed list of d.* commands passed to d.multicall2 to
// populate a Torrent, in the order Torrent's fields are read by
// torrentFromRow.
//
// Column order here must exactly match the read order in torrentFromRow.
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
	"d.custom1=",
	"d.custom2=",
	"d.custom3=",
	"d.custom4=",
	"d.custom5=",
	"d.hashing=",
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
	// Custom1 is the torrent's first custom field (commonly used as a label).
	Custom1 string
	// Custom2 is the torrent's second custom field.
	Custom2 string
	// Custom3 is the torrent's third custom field.
	Custom3 string
	// Custom4 is the torrent's fourth custom field.
	Custom4 string
	// Custom5 is the torrent's fifth custom field.
	Custom5 string
	// Hashing reports the torrent's hashing status:
	// 0 (not hashing),
	// 1 (initial hash check),
	// 2 (hash check on download completion),
	// 3 (rehash requested by the user).
	Hashing int64
}

// torrentFromRow converts one d.multicall2 result row into a Torrent. Column
// order must match torrentColumns.
func torrentFromRow(row []Value) (*Torrent, error) {
	var t Torrent
	r := newRowReader("torrent", row, len(torrentColumns))
	r.readString(&t.Hash, "hash")
	r.readString(&t.Name, "name")
	r.readInt64(&t.SizeBytes, "size bytes")
	r.readInt64(&t.CompletedBytes, "completed bytes")
	r.readInt64(&t.LeftBytes, "left bytes")
	r.readInt64(&t.DownRate, "down rate")
	r.readInt64(&t.UpRate, "up rate")
	r.readInt64(&t.DownTotal, "down total")
	r.readInt64(&t.UpTotal, "up total")
	r.readInt64(&t.Ratio, "ratio")
	r.readInt64(&t.State, "state")
	r.readBool(&t.IsActive, "is active")
	r.readBool(&t.IsOpen, "is open")
	r.readBool(&t.IsMultiFile, "is multi file")
	r.readBool(&t.IsPrivate, "is private")
	r.readString(&t.Message, "message")
	r.readString(&t.BasePath, "base path")
	r.readString(&t.Directory, "directory")
	r.readInt64(&t.Priority, "priority")
	r.readString(&t.Custom1, "custom1")
	r.readString(&t.Custom2, "custom2")
	r.readString(&t.Custom3, "custom3")
	r.readString(&t.Custom4, "custom4")
	r.readString(&t.Custom5, "custom5")
	r.readInt64(&t.Hashing, "hashing")
	if r.err != nil {
		return nil, r.err
	}
	return &t, nil
}

// Torrents returns the torrents visible in view, populated via d.multicall2.
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

// TorrentsCustom calls d.multicall2 against view with cmds, returning one raw
// row of Values per torrent.
func (c *Client) TorrentsCustom(ctx context.Context, view string, cmds ...string) ([][]Value, error) {
	leading := []Value{NewString(""), NewString(view)}
	return c.Multicall(ctx, "d.multicall2", leading, cmds...)
}
