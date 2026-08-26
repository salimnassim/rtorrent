package rtorrent

import (
	"context"
	"testing"
)

func TestClientFilesEndToEnd(t *testing.T) {
	const resp = `<?xml version="1.0"?><methodResponse><params><param><value><array><data>` +
		`<value><array><data>` +
		`<value><string>ubuntu.iso</string></value>` + // f.path=
		`<value><i8>1000000000</i8></value>` + // f.size_bytes=
		`<value><i8>238</i8></value>` + // f.size_chunks=
		`<value><i8>119</i8></value>` + // f.completed_chunks=
		`<value><i8>1</i8></value>` + // f.priority=
		`<value><i8>1</i8></value>` + // f.is_created=
		`<value><i8>0</i8></value>` + // f.offset=
		`</data></array></value>` +
		`</data></array></value></param></params></methodResponse>`

	c := newClient(&stubTransport{
		callFunc: func(ctx context.Context, body []byte) ([]byte, error) {
			return []byte(resp), nil
		},
	}, nil)

	files, err := c.Files(context.Background(), "0123456789ABCDEF0123456789ABCDEF01234567")
	if err != nil {
		t.Fatalf("Files() unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("Files() returned %d files, want 1", len(files))
	}

	got := files[0]
	want := &File{
		Path:            "ubuntu.iso",
		SizeBytes:       1000000000,
		SizeChunks:      238,
		CompletedChunks: 119,
		Priority:        1,
		IsCreated:       true,
		Offset:          0,
	}

	if *got != *want {
		t.Errorf("Files()[0] = %+v, want %+v", got, want)
	}
}

func TestFileFromRowColumnMismatch(t *testing.T) {
	_, err := fileFromRow([]Value{NewString("only one column")})
	if err == nil {
		t.Fatal("fileFromRow() error = nil, want error for column count mismatch")
	}
}
