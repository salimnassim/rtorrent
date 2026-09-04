package rtorrent

import (
	"bytes"
	"encoding/base64"
	"sort"
	"strconv"
	"strings"
)

// encodeMethodCall serializes an XML-RPC methodCall.
func encodeMethodCall(name string, params []Value) []byte {
	var buf bytes.Buffer

	buf.WriteString(`<?xml version="1.0"?>`)
	buf.WriteString("<methodCall><methodName>")
	buf.WriteString(escapeXML(name))
	buf.WriteString("</methodName><params>")

	for _, p := range params {
		buf.WriteString("<param>")
		encodeValue(&buf, p)
		buf.WriteString("</param>")
	}

	buf.WriteString("</params></methodCall>")

	return buf.Bytes()
}

// encodeValue writes v to buf as an XML-RPC <value> element. KindInt64
// emits <i8>, not <i4>.
func encodeValue(buf *bytes.Buffer, v Value) {
	buf.WriteString("<value>")

	switch v.Kind() {
	case KindString:
		buf.WriteString("<string>")
		buf.WriteString(escapeXML(v.str))
		buf.WriteString("</string>")

	case KindInt:
		buf.WriteString("<i4>")
		buf.WriteString(strconv.FormatInt(v.num, 10))
		buf.WriteString("</i4>")

	case KindInt64:
		buf.WriteString("<i8>")
		buf.WriteString(strconv.FormatInt(v.num, 10))
		buf.WriteString("</i8>")

	case KindDouble:
		buf.WriteString("<double>")
		buf.WriteString(strconv.FormatFloat(v.dbl, 'f', -1, 64))
		buf.WriteString("</double>")

	case KindBool:
		buf.WriteString("<boolean>")
		if v.b {
			buf.WriteString("1")
		} else {
			buf.WriteString("0")
		}
		buf.WriteString("</boolean>")

	case KindNil:
		buf.WriteString("<nil/>")

	case KindArray:
		buf.WriteString("<array><data>")
		for _, e := range v.arr {
			encodeValue(buf, e)
		}
		buf.WriteString("</data></array>")

	case KindStruct:
		buf.WriteString("<struct>")

		// Sorted for deterministic wire output.
		keys := make([]string, 0, len(v.strct))
		for k := range v.strct {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			buf.WriteString("<member><name>")
			buf.WriteString(escapeXML(k))
			buf.WriteString("</name>")
			encodeValue(buf, v.strct[k])
			buf.WriteString("</member>")
		}
		buf.WriteString("</struct>")

	case KindBase64:
		buf.WriteString("<base64>")
		buf.WriteString(base64.StdEncoding.EncodeToString([]byte(v.str)))
		buf.WriteString("</base64>")
	}

	buf.WriteString("</value>")
}

// escapeXML escapes the characters that are unsafe in XML element text
// content, and any byte below 0x20 (other than tab and LF) that XML 1.0
// does not allow to appear literally anywhere in a document. It is only
// ever used for element text, never attribute values, so "\"" and "'" are
// deliberately left unescaped.
func escapeXML(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '&':
			b.WriteString("&amp;")
		case c == '<':
			b.WriteString("&lt;")
		case c == '>':
			b.WriteString("&gt;")
		case c == '\r':
			b.WriteString("&#13;")
		case c < 0x20 && c != '\t' && c != '\n':
			b.WriteString("&#")
			b.WriteString(strconv.Itoa(int(c)))
			b.WriteByte(';')
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}
