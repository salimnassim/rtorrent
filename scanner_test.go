package rtorrent

import (
	"errors"
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func scanAll(t *testing.T, data string) ([]xmlToken, error) {
	t.Helper()
	s := newXMLScanner([]byte(data))
	var toks []xmlToken
	for {
		tok, err := s.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return toks, nil
			}
			return toks, err
		}
		toks = append(toks, tok)
	}
}

func TestXMLScannerToken(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []xmlToken
	}{
		{
			name: "start and end tag",
			in:   `<foo></foo>`,
			want: []xmlToken{
				{kind: tokenStart, name: "foo"},
				{kind: tokenEnd, name: "foo"},
			},
		},
		{
			name: "self-closing tag",
			in:   `<nil/>`,
			want: []xmlToken{
				{kind: tokenStart, name: "nil"},
				{kind: tokenEnd, name: "nil"},
			},
		},
		{
			name: "self-closing tag with space before slash",
			in:   `<nil />`,
			want: []xmlToken{
				{kind: tokenStart, name: "nil"},
				{kind: tokenEnd, name: "nil"},
			},
		},
		{
			name: "text content",
			in:   `<a>hellå</a>`,
			want: []xmlToken{
				{kind: tokenStart, name: "a"},
				{kind: tokenText, text: []byte("hellå")},
				{kind: tokenEnd, name: "a"},
			},
		},
		{
			name: "processing instruction is skipped",
			in:   `<?xml version="1.0"?><a></a>`,
			want: []xmlToken{
				{kind: tokenStart, name: "a"},
				{kind: tokenEnd, name: "a"},
			},
		},
		{
			name: "comment is skipped",
			in:   `<!-- a comment --><a></a>`,
			want: []xmlToken{
				{kind: tokenStart, name: "a"},
				{kind: tokenEnd, name: "a"},
			},
		},
		{
			name: "doctype-like declaration is skipped",
			in:   `<!DOCTYPE foo><a></a>`,
			want: []xmlToken{
				{kind: tokenStart, name: "a"},
				{kind: tokenEnd, name: "a"},
			},
		},
		{
			name: "cdata section passes through without unescaping",
			in:   `<a><![CDATA[raw & <text>]]></a>`,
			want: []xmlToken{
				{kind: tokenStart, name: "a"},
				{kind: tokenText, text: []byte("raw & <text>")},
				{kind: tokenEnd, name: "a"},
			},
		},
		{
			name: "predefined entities",
			in:   `&amp;&lt;&gt;&quot;&apos;`,
			want: []xmlToken{
				{kind: tokenText, text: []byte(`&<>"'`)},
			},
		},
		{
			name: "numeric character references",
			in:   `&#65;&#x42;&#X43;`,
			want: []xmlToken{
				{kind: tokenText, text: []byte("ABC")},
			},
		},
		{
			name: "nested elements",
			in:   `<a><b>x</b><c/></a>`,
			want: []xmlToken{
				{kind: tokenStart, name: "a"},
				{kind: tokenStart, name: "b"},
				{kind: tokenText, text: []byte("x")},
				{kind: tokenEnd, name: "b"},
				{kind: tokenStart, name: "c"},
				{kind: tokenEnd, name: "c"},
				{kind: tokenEnd, name: "a"},
			},
		},
		{
			name: "trailing whitespace before close angle, no attributes",
			in:   `<foo ></foo>`,
			want: []xmlToken{
				{kind: tokenStart, name: "foo"},
				{kind: tokenEnd, name: "foo"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := scanAll(t, tt.in)
			if err != nil {
				t.Fatalf("scanAll(%q) unexpected error: %v", tt.in, err)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(xmlToken{})); diff != "" {
				t.Errorf("scanAll(%q) mismatch (-want +got):\n%s", tt.in, diff)
			}
		})
	}
}

func TestXMLScannerTokenErrors(t *testing.T) {
	tests := []struct {
		name string
		in   string
	}{
		{name: "unterminated start tag", in: `<foo`},
		{name: "unterminated start tag no close angle", in: `<foo/`},
		{name: "unterminated end tag", in: `</foo`},
		{name: "unterminated processing instruction", in: `<?xml version="1.0"`},
		{name: "unterminated comment", in: `<!-- oops`},
		{name: "unterminated doctype-like declaration", in: `<!DOCTYPE foo`},
		{name: "unterminated cdata", in: `<![CDATA[oops`},
		{name: "empty tag name", in: `<>`},
		{name: "unterminated entity reference", in: `&amp`},
		{name: "unknown entity reference", in: `&bogus;`},
		{name: "invalid numeric character reference", in: `&#zz;`},
		{name: "invalid hex character reference", in: `&#xzz;`},
		{name: "start tag with attribute containing a slash", in: `<value foo="a/b">x</value>`},
		{name: "start tag with attribute containing a close angle", in: `<value foo="a>b">x</value>`},
		{name: "malformed self-closing tag", in: `<foo/x>`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := scanAll(t, tt.in); err == nil {
				t.Fatalf("scanAll(%q) error = nil, want error", tt.in)
			}
		})
	}
}

func FuzzXMLScannerToken(f *testing.F) {
	seeds := []string{
		`<foo></foo>`,
		`<nil/>`,
		`<a>hello</a>`,
		`<?xml version="1.0"?><a></a>`,
		`<!-- a comment --><a></a>`,
		`<!DOCTYPE foo><a></a>`,
		`<a><![CDATA[raw & <text>]]></a>`,
		`&amp;&lt;&gt;&quot;&apos;`,
		`&#65;&#x42;`,
		`<a><b>x</b><c/></a>`,
		`<foo`,
		`</foo`,
		`<?xml version="1.0"`,
		`<!-- asdfg`,
		`<![CDATA[cvbnöä`,
		`&amp`,
		`&bogus;`,
	}
	for _, s := range seeds {
		f.Add([]byte(s))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		s := newXMLScanner(data)
		for i := 0; i < 10_000; i++ {
			if _, err := s.Token(); err != nil {
				return
			}
		}
	})
}
