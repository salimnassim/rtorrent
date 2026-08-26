package rtorrent

import (
	"context"
	"fmt"
)

// fileColumns is the fixed list of f.* commands passed to f.multicall to
// populate a File, in the order File's fields are read by fileFromRow.
var fileColumns = []string{
	"f.path=",
	"f.size_bytes=",
	"f.size_chunks=",
	"f.completed_chunks=",
	"f.priority=",
	"f.is_created=",
	"f.offset=",
}

// File is a snapshot of one file within a torrent.
type File struct {
	// Path is the file's path, relative to the torrent's base path.
	Path string
	// SizeBytes is the file's total size.
	SizeBytes int64
	// SizeChunks is the number of chunks the file spans.
	SizeChunks int64
	// CompletedChunks is the number of the file's chunks downloaded and
	// verified so far.
	CompletedChunks int64
	// Priority is 0 (off), 1 (normal), or 2 (high).
	Priority int64
	// IsCreated reports whether the file has been created on disk.
	IsCreated bool
	// Offset is the file's byte offset within the torrent's concatenated
	// data.
	Offset int64
}

// fileFromRow converts one f.multicall result row into a File.
func fileFromRow(row []Value) (*File, error) {
	if len(row) != len(fileColumns) {
		return nil, fmt.Errorf("rtorrent: file row: got %d columns, want %d", len(row), len(fileColumns))
	}

	var (
		f   File
		err error
	)
	if f.Path, err = row[0].AsString(); err != nil {
		return nil, fmt.Errorf("rtorrent: file row: path: %w", err)
	}
	if f.SizeBytes, err = row[1].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: file row: size bytes: %w", err)
	}
	if f.SizeChunks, err = row[2].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: file row: size chunks: %w", err)
	}
	if f.CompletedChunks, err = row[3].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: file row: completed chunks: %w", err)
	}
	if f.Priority, err = row[4].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: file row: priority: %w", err)
	}

	isCreated, err := row[5].AsInt64()
	if err != nil {
		return nil, fmt.Errorf("rtorrent: file row: is created: %w", err)
	}
	f.IsCreated = isCreated != 0

	if f.Offset, err = row[6].AsInt64(); err != nil {
		return nil, fmt.Errorf("rtorrent: file row: offset: %w", err)
	}

	return &f, nil
}

// Files returns the files in the torrent identified by hash.
func (c *Client) Files(ctx context.Context, hash string) ([]*File, error) {
	rows, err := c.FilesCustom(ctx, hash, fileColumns...)
	if err != nil {
		return nil, err
	}

	files := make([]*File, len(rows))
	for i, row := range rows {
		f, err := fileFromRow(row)
		if err != nil {
			return nil, err
		}
		files[i] = f
	}
	return files, nil
}

// FilesCustom calls f.multicall against the torrent identified by hash with
// cmds, returning one raw row of Values per file.
func (c *Client) FilesCustom(ctx context.Context, hash string, cmds ...string) ([][]Value, error) {
	leading := []Value{NewString(hash), NewString("")}
	return c.Multicall(ctx, "f.multicall", leading, cmds...)
}
