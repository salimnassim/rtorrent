package rtorrent

import (
	"context"
	"fmt"
)

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
	if len(row) != len(peerColumns) {
		return nil, fmt.Errorf("rtorrent: peer row: got %d columns, want %d", len(row), len(peerColumns))
	}

	var (
		p   Peer
		err error
	)
	if p.ID, err = row[0].AsString(); err != nil {
		return nil, fmt.Errorf("rtorrent: peer row: id: %w", err)
	}
	if p.Address, err = row[1].AsString(); err != nil {
		return nil, fmt.Errorf("rtorrent: peer row: address: %w", err)
	}
	if p.Port, err = row[2].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: peer row: port: %w", err)
	}
	if p.ClientVersion, err = row[3].AsString(); err != nil {
		return nil, fmt.Errorf("rtorrent: peer row: client version: %w", err)
	}

	isEncrypted, err := row[4].AsInt64()
	if err != nil {
		return nil, fmt.Errorf("rtorrent: peer row: is encrypted: %w", err)
	}
	p.IsEncrypted = isEncrypted != 0

	isIncoming, err := row[5].AsInt64()
	if err != nil {
		return nil, fmt.Errorf("rtorrent: peer row: is incoming: %w", err)
	}
	p.IsIncoming = isIncoming != 0

	if p.UpRate, err = row[6].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: peer row: up rate: %w", err)
	}
	if p.UpTotal, err = row[7].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: peer row: up total: %w", err)
	}
	if p.DownRate, err = row[8].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: peer row: down rate: %w", err)
	}
	if p.DownTotal, err = row[9].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: peer row: down total: %w", err)
	}
	if p.CompletedPercent, err = row[10].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: peer row: completed percent: %w", err)
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
