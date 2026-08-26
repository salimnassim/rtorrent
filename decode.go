package rtorrent

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// decodeMethodResponse parses an XML-RPC methodResponse document and
// returns its single return value. If the response is a <fault>, it
// returns a *Fault as the error, reachable via errors.As.
func decodeMethodResponse(data []byte) (Value, error) {
	p := &rpcParser{dec: xml.NewDecoder(bytes.NewReader(data))}

	se, err := p.nextStart()
	if err != nil {
		return Value{}, err
	}
	if se.Name.Local != "methodResponse" {
		return Value{}, fmt.Errorf("decode xml-rpc: expected methodResponse element, got %q", se.Name.Local)
	}

	se, err = p.nextStart()
	if err != nil {
		return Value{}, err
	}

	switch se.Name.Local {
	case "params":
		if se, err = p.nextStart(); err != nil {
			return Value{}, err
		}
		if se.Name.Local != "param" {
			return Value{}, fmt.Errorf("decode xml-rpc: expected param element, got %q", se.Name.Local)
		}
		if se, err = p.nextStart(); err != nil {
			return Value{}, err
		}
		if se.Name.Local != "value" {
			return Value{}, fmt.Errorf("decode xml-rpc: expected value element, got %q", se.Name.Local)
		}
		return p.parseValue()

	case "fault":
		if se, err = p.nextStart(); err != nil {
			return Value{}, err
		}
		if se.Name.Local != "value" {
			return Value{}, fmt.Errorf("decode xml-rpc: expected value element in fault, got %q", se.Name.Local)
		}
		v, err := p.parseValue()
		if err != nil {
			return Value{}, err
		}
		f, err := faultFromValue(v)
		if err != nil {
			return Value{}, err
		}
		return Value{}, f

	default:
		return Value{}, fmt.Errorf("decode xml-rpc: expected params or fault element, got %q", se.Name.Local)
	}
}

// rpcParser is a hand-written recursive-descent parser over an
// encoding/xml.Decoder token stream. encoding/xml is used only as a
// tokenizer here: the XML-RPC value tree's shape depends on which tag is
// encountered while descending, which xml.Unmarshal into tagged structs
// handles awkwardly for a recursive tagged union like this one.
type rpcParser struct {
	dec *xml.Decoder
}

// nextStart advances past any non-element tokens and returns the next
// start element.
func (p *rpcParser) nextStart() (xml.StartElement, error) {
	for {
		tok, err := p.dec.Token()
		if err != nil {
			return xml.StartElement{}, fmt.Errorf("decode xml-rpc: %w", err)
		}
		if se, ok := tok.(xml.StartElement); ok {
			return se, nil
		}
	}
}

// expectEnd consumes the next token and requires it to be an end element
// named name.
func (p *rpcParser) expectEnd(name string) error {
	tok, err := p.dec.Token()
	if err != nil {
		return fmt.Errorf("decode xml-rpc: %w", err)
	}
	end, ok := tok.(xml.EndElement)
	if !ok || end.Name.Local != name {
		return fmt.Errorf("decode xml-rpc: expected end element %q", name)
	}
	return nil
}

// readText reads character data up to and including the end element named
// name. It is only safe for leaf scalar tags that the XML-RPC spec never
// nests same-named tags inside of (string, i4, int, i8, double, boolean,
// nil, name); it does not depth-track, so a nested element with the same
// name would otherwise cause it to stop early and return truncated data.
// A self-closing tag (e.g. <name/>) arrives as a start element immediately
// followed by its end element with no intervening CharData, which this
// loop handles naturally by returning an empty string.
func (p *rpcParser) readText(name string) (string, error) {
	var sb strings.Builder
	for {
		tok, err := p.dec.Token()
		if err != nil {
			return "", fmt.Errorf("decode xml-rpc: %w", err)
		}
		switch t := tok.(type) {
		case xml.CharData:
			sb.Write(t)
		case xml.EndElement:
			if t.Name.Local != name {
				return "", fmt.Errorf("decode xml-rpc: expected end element %q, got %q", name, t.Name.Local)
			}
			return sb.String(), nil
		}
	}
}

// parseValue parses the content of a <value> element whose start tag has
// already been consumed by the caller, and consumes through its matching
// end tag before returning.
func (p *rpcParser) parseValue() (Value, error) {
	for {
		tok, err := p.dec.Token()
		if err != nil {
			return Value{}, fmt.Errorf("decode xml-rpc: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			v, err := p.parseTypedValue(t)
			if err != nil {
				return Value{}, err
			}
			if err := p.expectEnd("value"); err != nil {
				return Value{}, err
			}
			return v, nil

		case xml.EndElement:
			// <value></value> with no type tag: an empty untyped value.
			return NewString(""), nil

		case xml.CharData:
			s := strings.TrimSpace(string(t))
			if s == "" {
				continue
			}
			// Untyped text directly inside <value> defaults to a string,
			// per the XML-RPC spec.
			if err := p.expectEnd("value"); err != nil {
				return Value{}, err
			}
			return NewString(s), nil
		}
	}
}

// parseTypedValue parses the element se as a typed XML-RPC value, consuming
// through se's own end tag.
func (p *rpcParser) parseTypedValue(se xml.StartElement) (Value, error) {
	name := se.Name.Local

	switch name {
	case "string":
		s, err := p.readText(name)
		if err != nil {
			return Value{}, err
		}
		return NewString(s), nil

	case "i4", "int":
		return p.parseInt(name, NewInt)

	case "i8":
		return p.parseInt(name, NewInt64)

	case "double":
		s, err := p.readText(name)
		if err != nil {
			return Value{}, err
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
		if err != nil {
			return Value{}, fmt.Errorf("decode xml-rpc: invalid double value %q: %w", s, err)
		}
		return NewDouble(f), nil

	case "boolean":
		s, err := p.readText(name)
		if err != nil {
			return Value{}, err
		}
		switch strings.TrimSpace(s) {
		case "0":
			return NewBool(false), nil
		case "1":
			return NewBool(true), nil
		default:
			return Value{}, fmt.Errorf("decode xml-rpc: invalid boolean value %q", s)
		}

	case "nil":
		if _, err := p.readText(name); err != nil {
			return Value{}, err
		}
		return NewNil(), nil

	case "array":
		return p.parseArray()

	case "struct":
		return p.parseStruct()

	default:
		return p.parseUnknown(se)
	}
}

// parseInt reads an integer scalar tag and wraps it with newValue.
func (p *rpcParser) parseInt(name string, newValue func(int64) Value) (Value, error) {
	s, err := p.readText(name)
	if err != nil {
		return Value{}, err
	}
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil {
		return Value{}, fmt.Errorf("decode xml-rpc: invalid %s value %q: %w", name, s, err)
	}
	return newValue(n), nil
}

// parseArray parses the content of an <array> element, consuming through
// its own end tag.
func (p *rpcParser) parseArray() (Value, error) {
	se, err := p.nextStart()
	if err != nil {
		return Value{}, err
	}
	if se.Name.Local != "data" {
		return Value{}, fmt.Errorf("decode xml-rpc: expected data element inside array, got %q", se.Name.Local)
	}

	var elems []Value
	for {
		tok, err := p.dec.Token()
		if err != nil {
			return Value{}, fmt.Errorf("decode xml-rpc: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local != "value" {
				return Value{}, fmt.Errorf("decode xml-rpc: unexpected element %q inside array data", t.Name.Local)
			}
			v, err := p.parseValue()
			if err != nil {
				return Value{}, err
			}
			elems = append(elems, v)

		case xml.EndElement:
			if t.Name.Local != "data" {
				return Value{}, fmt.Errorf("decode xml-rpc: expected end of data, got %q", t.Name.Local)
			}
			if err := p.expectEnd("array"); err != nil {
				return Value{}, err
			}
			return NewArray(elems), nil
		}
	}
}

// parseStruct parses the content of a <struct> element, consuming through
// its own end tag.
func (p *rpcParser) parseStruct() (Value, error) {
	members := make(map[string]Value)
	for {
		tok, err := p.dec.Token()
		if err != nil {
			return Value{}, fmt.Errorf("decode xml-rpc: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local != "member" {
				return Value{}, fmt.Errorf("decode xml-rpc: unexpected element %q inside struct", t.Name.Local)
			}
			name, value, err := p.parseMember()
			if err != nil {
				return Value{}, err
			}
			members[name] = value

		case xml.EndElement:
			if t.Name.Local != "struct" {
				return Value{}, fmt.Errorf("decode xml-rpc: expected end of struct, got %q", t.Name.Local)
			}
			return NewStruct(members), nil
		}
	}
}

// parseMember parses a <member> element's <name> and <value> children,
// consuming through the member's own end tag.
func (p *rpcParser) parseMember() (string, Value, error) {
	se, err := p.nextStart()
	if err != nil {
		return "", Value{}, err
	}
	if se.Name.Local != "name" {
		return "", Value{}, fmt.Errorf("decode xml-rpc: expected name element in struct member, got %q", se.Name.Local)
	}
	name, err := p.readText("name")
	if err != nil {
		return "", Value{}, err
	}

	if se, err = p.nextStart(); err != nil {
		return "", Value{}, err
	}
	if se.Name.Local != "value" {
		return "", Value{}, fmt.Errorf("decode xml-rpc: expected value element in struct member, got %q", se.Name.Local)
	}
	value, err := p.parseValue()
	if err != nil {
		return "", Value{}, err
	}

	if err := p.expectEnd("member"); err != nil {
		return "", Value{}, err
	}
	return name, value, nil
}

// parseUnknown handles a tag not recognized as one of the standard
// XML-RPC types, consuming through its own end tag with the same
// depth-tracking a structured skip needs. An unrecognized tag with no
// child elements degrades to a string value (forward-compatible with new
// scalar-like extensions). An unrecognized tag containing child elements
// returns an explicit error instead of being read as a plausible-looking
// but silently truncated or corrupted value.
func (p *rpcParser) parseUnknown(se xml.StartElement) (Value, error) {
	name := se.Name.Local
	depth := 0
	hasChild := false
	var text strings.Builder

	for {
		tok, err := p.dec.Token()
		if err != nil {
			return Value{}, fmt.Errorf("decode xml-rpc: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			hasChild = true
			depth++

		case xml.EndElement:
			if depth == 0 {
				if hasChild {
					return Value{}, fmt.Errorf("unsupported XML-RPC type %q", name)
				}
				return NewString(text.String()), nil
			}
			depth--

		case xml.CharData:
			if depth == 0 {
				text.Write(t)
			}
		}
	}
}
