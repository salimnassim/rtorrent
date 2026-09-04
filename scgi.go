package rtorrent

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
)

// scgiTransport sends XML-RPC requests directly to the SCGI listener.
type scgiTransport struct {
	// network is "tcp" or "unix".
	network string
	address string
}

// call implements transport.
func (s *scgiTransport) call(ctx context.Context, body []byte) ([]byte, error) {
	var d net.Dialer
	conn, err := d.DialContext(ctx, s.network, s.address)
	if err != nil {
		return nil, fmt.Errorf("scgi: dial %s %s: %w", s.network, s.address, err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		if err := conn.SetDeadline(deadline); err != nil {
			return nil, fmt.Errorf("scgi: set deadline: %w", err)
		}
	}

	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()

	if _, err := conn.Write(scgiRequest(body)); err != nil {
		return nil, fmt.Errorf("scgi: write request: %w", err)
	}

	respBody, err := readSCGIResponse(conn)
	if err != nil {
		return nil, fmt.Errorf("scgi: %w", err)
	}
	return respBody, nil
}

// scgiRequest frames body as an SCGI request: a netstring-encoded header
// block (NUL-separated key\0value\0 pairs, CONTENT_LENGTH first, then the
// SCGI=1 header).
func scgiRequest(body []byte) []byte {
	var headers bytes.Buffer
	writeSCGIHeader(&headers, "CONTENT_LENGTH", strconv.Itoa(len(body)))
	writeSCGIHeader(&headers, "SCGI", "1")

	var buf bytes.Buffer
	buf.WriteString(strconv.Itoa(headers.Len()))
	buf.WriteByte(':')
	buf.Write(headers.Bytes())
	buf.WriteByte(',')
	buf.Write(body)
	return buf.Bytes()
}

func writeSCGIHeader(buf *bytes.Buffer, key, value string) {
	buf.WriteString(key)
	buf.WriteByte(0)
	buf.WriteString(value)
	buf.WriteByte(0)
}

const (
	maxSCGIHeaderBytes   = 8 << 10
	maxSCGIHeaderLines   = 64
	maxSCGIResponseBytes = 8 << 20
)

// readSCGIResponse reads an SCGI response from r: CGI-style "Header: value"
// lines up to a blank line, then the body.
func readSCGIResponse(r io.Reader) ([]byte, error) {
	br := bufio.NewReader(io.LimitReader(r, maxSCGIHeaderBytes+1))
	headerBytes := 0
	lines := 0
	for {
		line, err := br.ReadString('\n')
		headerBytes += len(line)
		if err != nil {
			if headerBytes > maxSCGIHeaderBytes {
				return nil, fmt.Errorf("response headers exceed %d bytes", maxSCGIHeaderBytes)
			}
			return nil, fmt.Errorf("truncated response headers: %w", err)
		}
		if strings.TrimRight(line, "\r\n") == "" {
			break
		}
		lines++
		if lines > maxSCGIHeaderLines {
			return nil, fmt.Errorf("response headers exceed %d lines", maxSCGIHeaderLines)
		}
	}

	respBody, err := io.ReadAll(io.LimitReader(io.MultiReader(br, r), maxSCGIResponseBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	if len(respBody) > maxSCGIResponseBytes {
		return nil, fmt.Errorf("response body exceeds %d bytes", maxSCGIResponseBytes)
	}
	return respBody, nil
}
