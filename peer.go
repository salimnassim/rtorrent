package rtorrent

import "context"

// peerColumns is the fixed list of p.* commands passed to p.multicall to
// populate a Peer, in the order Peer's fields are read by peerFromRow.
var peerColumns = []string{
	"p.id=",
	"p.address=",
	"p.port=",
	"p.client_version=",
	"p.is_encrypted=",
	"p.is_incoming=",
	"p.up_rate=",
	"p.up_total=",
	"p.down_rate=",
	"p.down_total=",
	"p.completed_percent=",
}

// Peer is a snapshot of one peer connection for a torrent.
type Peer struct {
	// ID is the peer's hex-encoded peer id.
	ID string
	// Address is the peer's IP address (IPv4 dotted-decimal, or IPv6 in
	// brackets).
	Address string
	// Port is the peer's port.
	Port int64
	// ClientVersion is the peer client's identified version string.
	ClientVersion string
	// IsEncrypted reports whether the connection is encrypted.
	IsEncrypted bool
	// IsIncoming reports whether the peer connected to us, rather than us
	// to it.
	IsIncoming bool
	// UpRate is the current upload rate to this peer, in bytes per second.
	UpRate int64
	// UpTotal is the total bytes uploaded to this peer.
	UpTotal int64
	// DownRate is the current download rate from this peer, in bytes per
	// second.
	DownRate int64
	// DownTotal is the total bytes downloaded from this peer.
	DownTotal int64
	// CompletedPercent is the percentage, 0-100, of the torrent this peer
	// has reported as complete.
	CompletedPercent int64
}

// peerFromRow converts one p.multicall result row into a Peer.
func peerFromRow(row []Value) (*Peer, error) {
	var p Peer
	r := newRowReader("peer", row, len(peerColumns))
	r.readString(&p.ID, "id")
	r.readString(&p.Address, "address")
	r.readInt64(&p.Port, "port")
	r.readString(&p.ClientVersion, "client version")
	r.readBool(&p.IsEncrypted, "is encrypted")
	r.readBool(&p.IsIncoming, "is incoming")
	r.readInt64(&p.UpRate, "up rate")
	r.readInt64(&p.UpTotal, "up total")
	r.readInt64(&p.DownRate, "down rate")
	r.readInt64(&p.DownTotal, "down total")
	r.readInt64(&p.CompletedPercent, "completed percent")
	if r.err != nil {
		return nil, r.err
	}
	return &p, nil
}

// Peers returns the connected peers for the torrent identified by hash.
func (c *Client) Peers(ctx context.Context, hash string) ([]*Peer, error) {
	rows, err := c.PeersCustom(ctx, hash, peerColumns...)
	if err != nil {
		return nil, err
	}

	peers := make([]*Peer, len(rows))
	for i, row := range rows {
		p, err := peerFromRow(row)
		if err != nil {
			return nil, err
		}
		peers[i] = p
	}
	return peers, nil
}

// PeersCustom calls p.multicall against the torrent identified by hash with
// cmds, returning one raw row of Values per peer.
func (c *Client) PeersCustom(ctx context.Context, hash string, cmds ...string) ([][]Value, error) {
	leading := []Value{NewString(hash), NewString("")}
	return c.Multicall(ctx, "p.multicall", leading, cmds...)
}
