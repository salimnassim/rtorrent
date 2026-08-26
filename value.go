package rtorrent

import (
	"errors"
	"fmt"
)

// Kind identifies the XML-RPC type by a Value.
type Kind int

const (
	KindString Kind = iota
	KindInt
	KindInt64
	KindDouble
	KindBool
	KindArray
	KindStruct
	KindNil
)

// String returns the name of k.
func (k Kind) String() string {
	switch k {
	case KindString:
		return "string"
	case KindInt:
		return "int"
	case KindInt64:
		return "int64"
	case KindDouble:
		return "double"
	case KindBool:
		return "bool"
	case KindArray:
		return "array"
	case KindStruct:
		return "struct"
	case KindNil:
		return "nil"
	default:
		return fmt.Sprintf("Kind(%d)", int(k))
	}
}

var ErrKind = errors.New("value: wrong kind")

// Value is a single XML-RPC value: a scalar, an array of Values, or a
// struct of named Values.
type Value struct {
	kind  Kind
	str   string
	num   int64
	dbl   float64
	b     bool
	arr   []Value
	strct map[string]Value
}

// NewString returns a Value of Kind KindString.
func NewString(s string) Value {
	return Value{kind: KindString, str: s}
}

// NewInt returns a Value of Kind KindInt.
func NewInt(n int64) Value {
	return Value{kind: KindInt, num: n}
}

// NewInt64 returns a Value of Kind KindInt64.
func NewInt64(n int64) Value {
	return Value{kind: KindInt64, num: n}
}

// NewDouble returns a Value of Kind KindDouble.
func NewDouble(f float64) Value {
	return Value{kind: KindDouble, dbl: f}
}

// NewBool returns a Value of Kind KindBool.
func NewBool(b bool) Value {
	return Value{kind: KindBool, b: b}
}

// NewArray returns a Value of Kind KindArray wrapping elems.
func NewArray(elems []Value) Value {
	return Value{kind: KindArray, arr: elems}
}

// NewStruct returns a Value of Kind KindStruct wrapping members.
func NewStruct(members map[string]Value) Value {
	return Value{kind: KindStruct, strct: members}
}

// NewNil returns a Value of Kind KindNil.
func NewNil() Value {
	return Value{kind: KindNil}
}

// Kind returns the Kind of v.
func (v Value) Kind() Kind {
	return v.kind
}

// AsString returns v's string value.
func (v Value) AsString() (string, error) {
	if v.kind != KindString {
		return "", fmt.Errorf("%w: want %s, have %s", ErrKind, KindString, v.kind)
	}
	return v.str, nil
}

// AsInt64 returns v's integer value as an int64.
func (v Value) AsInt64() (int64, error) {
	switch v.kind {
	case KindInt, KindInt64:
		return v.num, nil
	default:
		return 0, fmt.Errorf("%w: want %s or %s, have %s", ErrKind, KindInt, KindInt64, v.kind)
	}
}

// AsDouble returns v's value as a float64..
func (v Value) AsDouble() (float64, error) {
	switch v.kind {
	case KindDouble:
		return v.dbl, nil
	case KindInt, KindInt64:
		return float64(v.num), nil
	default:
		return 0, fmt.Errorf("%w: want %s, %s, or %s, have %s", ErrKind, KindDouble, KindInt, KindInt64, v.kind)
	}
}

// AsBool returns v's boolean value.
func (v Value) AsBool() (bool, error) {
	if v.kind != KindBool {
		return false, fmt.Errorf("%w: want %s, have %s", ErrKind, KindBool, v.kind)
	}
	return v.b, nil
}

// AsArray returns v's elements.
func (v Value) AsArray() ([]Value, error) {
	if v.kind != KindArray {
		return nil, fmt.Errorf("%w: want %s, have %s", ErrKind, KindArray, v.kind)
	}
	return v.arr, nil
}

// AsStruct returns v's members.
func (v Value) AsStruct() (map[string]Value, error) {
	if v.kind != KindStruct {
		return nil, fmt.Errorf("%w: want %s, have %s", ErrKind, KindStruct, v.kind)
	}
	return v.strct, nil
}
