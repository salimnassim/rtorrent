package rtorrent

import "context"

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
	var f File
	r := newRowReader("file", row, len(fileColumns))
	r.readString(&f.Path, "path")
	r.readInt64(&f.SizeBytes, "size bytes")
	r.readInt64(&f.SizeChunks, "size chunks")
	r.readInt64(&f.CompletedChunks, "completed chunks")
	r.readInt64(&f.Priority, "priority")
	r.readBool(&f.IsCreated, "is created")
	r.readInt64(&f.Offset, "offset")
	if r.err != nil {
		return nil, r.err
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
