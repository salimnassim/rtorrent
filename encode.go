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
// content.
func escapeXML(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '&':
			b.WriteString("&amp;")
		case '<':
			b.WriteString("&lt;")
		case '>':
			b.WriteString("&gt;")
		case '\r':
			b.WriteString("&#13;")
		default:
			b.WriteByte(s[i])
		}
	}
	return b.String()
}
