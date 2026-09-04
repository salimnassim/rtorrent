package rtorrent

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// xmlTokenKind identifies the kind of xmlToken.
type xmlTokenKind int

const (
	tokenStart xmlTokenKind = iota
	tokenEnd
	tokenText
)

// xmlToken is one token.
type xmlToken struct {
	kind xmlTokenKind
	name string
	text []byte
}

// xmlScanner tokenizes a document directly from a byte slice, it
// implements only the subset of XML that is required.
type xmlScanner struct {
	data    []byte
	pos     int
	pending *xmlToken
}

// newXMLScanner returns a scanner over data.
func newXMLScanner(data []byte) *xmlScanner {
	return &xmlScanner{data: data}
}

// Token returns the next token in the stream, or an error wrapping io.EOF
// once the input is exhausted.
func (s *xmlScanner) Token() (xmlToken, error) {
	if s.pending != nil {
		t := *s.pending
		s.pending = nil
		return t, nil
	}

	for {
		if s.pos >= len(s.data) {
			return xmlToken{}, io.EOF
		}
		if s.data[s.pos] != '<' {
			return s.scanText()
		}

		rest := s.data[s.pos:]
		switch {
		case bytes.HasPrefix(rest, []byte("<?")):
			if err := s.skipTo("?>"); err != nil {
				return xmlToken{}, err
			}
		case bytes.HasPrefix(rest, []byte("<!--")):
			if err := s.skipTo("-->"); err != nil {
				return xmlToken{}, err
			}
		case bytes.HasPrefix(rest, []byte("<![CDATA[")):
			return s.scanCDATA()
		case bytes.HasPrefix(rest, []byte("<!")):
			if err := s.skipTo(">"); err != nil {
				return xmlToken{}, err
			}
		case bytes.HasPrefix(rest, []byte("</")):
			return s.scanEndTag()
		default:
			return s.scanStartTag()
		}
	}
}

// skipTo advances past the first occurrence of closer.
func (s *xmlScanner) skipTo(closer string) error {
	i := bytes.Index(s.data[s.pos:], []byte(closer))
	if i < 0 {
		return fmt.Errorf("decode xml-rpc: unterminated %q", closer)
	}
	s.pos += i + len(closer)
	return nil
}

// scanText scans a run of character data up to the next '<' or end of
// input.
func (s *xmlScanner) scanText() (xmlToken, error) {
	start := s.pos
	if i := bytes.IndexByte(s.data[s.pos:], '<'); i >= 0 {
		s.pos += i
	} else {
		s.pos = len(s.data)
	}
	text, err := unescapeText(s.data[start:s.pos])
	if err != nil {
		return xmlToken{}, err
	}
	return xmlToken{kind: tokenText, text: text}, nil
}

// scanCDATA scans a <![CDATA[...]]> section.
func (s *xmlScanner) scanCDATA() (xmlToken, error) {
	const open = "<![CDATA["
	const close = "]]>"
	contentStart := s.pos + len(open)
	i := bytes.Index(s.data[contentStart:], []byte(close))
	if i < 0 {
		return xmlToken{}, fmt.Errorf("decode xml-rpc: unterminated CDATA section")
	}
	text := s.data[contentStart : contentStart+i]
	s.pos = contentStart + i + len(close)
	return xmlToken{kind: tokenText, text: text}, nil
}

// scanEndTag scans a </name> end tag.
func (s *xmlScanner) scanEndTag() (xmlToken, error) {
	s.pos += len("</")
	start := s.pos
	for s.pos < len(s.data) && s.data[s.pos] != '>' {
		s.pos++
	}
	if s.pos >= len(s.data) {
		return xmlToken{}, fmt.Errorf("decode xml-rpc: unterminated end tag")
	}
	name := strings.TrimSpace(string(s.data[start:s.pos]))
	s.pos++ // consume '>'
	return xmlToken{kind: tokenEnd, name: name}, nil
}

// scanStartTag scans a <name> or self-closing <name/> start tag. XML-RPC
// elements never carry attributes, so a start tag whose name is followed by
// anything but optional whitespace and then '>' or '/>' is rejected outright
// rather than scanned past that avoids misreading a '/' inside a quoted
// attribute value as the self-closing marker.
func (s *xmlScanner) scanStartTag() (xmlToken, error) {
	s.pos++ // consume '<'
	start := s.pos
	for s.pos < len(s.data) && !isNameEnd(s.data[s.pos]) {
		s.pos++
	}
	if s.pos >= len(s.data) {
		return xmlToken{}, fmt.Errorf("decode xml-rpc: unterminated start tag")
	}
	name := string(s.data[start:s.pos])
	if name == "" {
		return xmlToken{}, fmt.Errorf("decode xml-rpc: empty tag name")
	}

	for s.pos < len(s.data) && isSpace(s.data[s.pos]) {
		s.pos++
	}
	if s.pos >= len(s.data) {
		return xmlToken{}, fmt.Errorf("decode xml-rpc: unterminated start tag %q", name)
	}

	selfClosing := false
	switch s.data[s.pos] {
	case '>':
	case '/':
		s.pos++
		if s.pos >= len(s.data) || s.data[s.pos] != '>' {
			return xmlToken{}, fmt.Errorf("decode xml-rpc: malformed start tag %q", name)
		}
		selfClosing = true
	default:
		return xmlToken{}, fmt.Errorf("decode xml-rpc: attributes are not supported in start tag %q", name)
	}
	s.pos++ // consume '>'

	if selfClosing {
		s.pending = &xmlToken{kind: tokenEnd, name: name}
	}
	return xmlToken{kind: tokenStart, name: name}, nil
}

// isNameEnd reports whether c terminates a tag name.
func isNameEnd(c byte) bool {
	switch c {
	case ' ', '\t', '\r', '\n', '/', '>':
		return true
	default:
		return false
	}
}

// isSpace reports whether c is XML whitespace.
func isSpace(c byte) bool {
	switch c {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}

// unescapeText decodes the five predefined XML entities and numeric
// character references in b.
func unescapeText(b []byte) ([]byte, error) {
	if bytes.IndexByte(b, '&') < 0 {
		return b, nil
	}

	var out bytes.Buffer
	out.Grow(len(b))
	for i := 0; i < len(b); {
		if b[i] != '&' {
			out.WriteByte(b[i])
			i++
			continue
		}
		semi := bytes.IndexByte(b[i:], ';')
		if semi < 0 {
			return nil, fmt.Errorf("decode xml-rpc: unterminated entity reference")
		}
		entity := string(b[i+1 : i+semi])
		switch {
		case entity == "amp":
			out.WriteByte('&')
		case entity == "lt":
			out.WriteByte('<')
		case entity == "gt":
			out.WriteByte('>')
		case entity == "quot":
			out.WriteByte('"')
		case entity == "apos":
			out.WriteByte('\'')
		case strings.HasPrefix(entity, "#x") || strings.HasPrefix(entity, "#X"):
			n, err := strconv.ParseInt(entity[2:], 16, 32)
			if err != nil {
				return nil, fmt.Errorf("decode xml-rpc: invalid numeric character reference %q: %w", entity, err)
			}
			out.WriteRune(rune(n))
		case strings.HasPrefix(entity, "#"):
			n, err := strconv.ParseInt(entity[1:], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("decode xml-rpc: invalid numeric character reference %q: %w", entity, err)
			}
			out.WriteRune(rune(n))
		default:
			return nil, fmt.Errorf("decode xml-rpc: unknown entity reference %q", entity)
		}
		i += semi + 1
	}
	return out.Bytes(), nil
}
