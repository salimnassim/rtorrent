package rtorrent

import (
	"bytes"
	"testing"
)

func TestEncodeValue(t *testing.T) {
	tests := []struct {
		name  string
		value Value
		want  string
	}{
		{name: "string", value: NewString("bar"), want: "<value><string>bar</string></value>"},
		{
			name:  "string escaping",
			value: NewString("<a & b>"),
			want:  "<value><string>&lt;a &amp; b&gt;</string></value>",
		},
		{name: "int", value: NewInt(42), want: "<value><i4>42</i4></value>"},
		{name: "negative int", value: NewInt(-7), want: "<value><i4>-7</i4></value>"},
		{
			name:  "int64 emits i8, not i4",
			value: NewInt64(9223372036854775807),
			want:  "<value><i8>9223372036854775807</i8></value>",
		},
		{name: "double", value: NewDouble(3.25), want: "<value><double>3.25</double></value>"},
		{name: "bool true", value: NewBool(true), want: "<value><boolean>1</boolean></value>"},
		{name: "bool false", value: NewBool(false), want: "<value><boolean>0</boolean></value>"},
		{name: "nil", value: NewNil(), want: "<value><nil/></value>"},
		{
			name:  "empty array",
			value: NewArray(nil),
			want:  "<value><array><data></data></array></value>",
		},
		{
			name:  "array",
			value: NewArray([]Value{NewInt(1), NewInt(2)}),
			want:  "<value><array><data><value><i4>1</i4></value><value><i4>2</i4></value></data></array></value>",
		},
		{
			name:  "struct single member",
			value: NewStruct(map[string]Value{"a": NewInt(1)}),
			want:  "<value><struct><member><name>a</name><value><i4>1</i4></value></member></struct></value>",
		},
		{
			name:  "struct sorts members for deterministic output",
			value: NewStruct(map[string]Value{"b": NewInt(2), "a": NewInt(1)}),
			want: "<value><struct>" +
				"<member><name>a</name><value><i4>1</i4></value></member>" +
				"<member><name>b</name><value><i4>2</i4></value></member>" +
				"</struct></value>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			encodeValue(&buf, tt.value)
			if got := buf.String(); got != tt.want {
				t.Errorf("encodeValue(%v) = %s, want %s", tt.value, got, tt.want)
			}
		})
	}
}

func TestEncodeMethodCall(t *testing.T) {
	got := string(encodeMethodCall("d.hash", []Value{NewString(""), NewString("d.hash=")}))
	want := `<?xml version="1.0"?>` +
		"<methodCall><methodName>d.hash</methodName><params>" +
		"<param><value><string></string></value></param>" +
		"<param><value><string>d.hash=</string></value></param>" +
		"</params></methodCall>"

	if got != want {
		t.Errorf("encodeMethodCall() = %s, want %s", got, want)
	}
}

func TestEncodeMethodCallNoParams(t *testing.T) {
	got := string(encodeMethodCall("system.listMethods", nil))
	want := `<?xml version="1.0"?>` +
		"<methodCall><methodName>system.listMethods</methodName><params></params></methodCall>"

	if got != want {
		t.Errorf("encodeMethodCall() = %s, want %s", got, want)
	}
}

func FuzzEncodeDecodeString(f *testing.F) {
	seeds := []string{
		"",
		"hello",
		"<a & b>",
		"line1\nline2",
		"tab\ttab",
		"\x00",
		"unicode ☃",
		string([]byte{0xff, 0xfe}),
		"carriage\rreturn",
		"crlf\r\nline",
	}
	for _, s := range seeds {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, s string) {
		var buf bytes.Buffer
		buf.WriteString("<methodResponse><params><param>")
		encodeValue(&buf, NewString(s))
		buf.WriteString("</param></params></methodResponse>")

		got, err := decodeMethodResponse(buf.Bytes())
		if err != nil {
			return
		}

		gotStr, err := got.AsString()
		if err != nil {
			t.Fatalf("decodeMethodResponse() returned non-string value: %v", err)
		}
		if gotStr != s {
			t.Errorf("round trip through encodeValue/decodeMethodResponse = %q, want %q", gotStr, s)
		}
	})
}
