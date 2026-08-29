package rtorrent

import (
	"bytes"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestValueAsString(t *testing.T) {
	tests := []struct {
		name    string
		value   Value
		want    string
		wantErr bool
	}{
		{name: "string", value: NewString("foo"), want: "foo"},
		{name: "empty string", value: NewString(""), want: ""},
		{name: "zero value", value: Value{}, want: ""},
		{name: "wrong kind int", value: NewInt(1), wantErr: true},
		{name: "wrong kind bool", value: NewBool(true), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.value.AsString()
			if tt.wantErr {
				if !errors.Is(err, ErrKind) {
					t.Errorf("AsString() error = %v, want error wrapping ErrKind", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("AsString() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("AsString() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValueAsInt64(t *testing.T) {
	tests := []struct {
		name    string
		value   Value
		want    int64
		wantErr bool
	}{
		{name: "int", value: NewInt(42), want: 42},
		{name: "int64", value: NewInt64(9223372036854775807), want: 9223372036854775807},
		{name: "negative", value: NewInt(-5), want: -5},
		{name: "wrong kind", value: NewString("42"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.value.AsInt64()
			if tt.wantErr {
				if !errors.Is(err, ErrKind) {
					t.Errorf("AsInt64() error = %v, want error wrapping ErrKind", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("AsInt64() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("AsInt64() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestValueAsDouble(t *testing.T) {
	tests := []struct {
		name    string
		value   Value
		want    float64
		wantErr bool
	}{
		{name: "double", value: NewDouble(3.5), want: 3.5},
		{name: "int widens", value: NewInt(2), want: 2},
		{name: "int64 widens", value: NewInt64(3), want: 3},
		{name: "wrong kind", value: NewString("3.5"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.value.AsDouble()
			if tt.wantErr {
				if !errors.Is(err, ErrKind) {
					t.Errorf("AsDouble() error = %v, want error wrapping ErrKind", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("AsDouble() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("AsDouble() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValueAsBool(t *testing.T) {
	tests := []struct {
		name    string
		value   Value
		want    bool
		wantErr bool
	}{
		{name: "true", value: NewBool(true), want: true},
		{name: "false", value: NewBool(false), want: false},
		{name: "does not guess from string", value: NewString("true"), wantErr: true},
		{name: "does not guess from int", value: NewInt(1), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.value.AsBool()
			if tt.wantErr {
				if !errors.Is(err, ErrKind) {
					t.Errorf("AsBool() error = %v, want error wrapping ErrKind", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("AsBool() unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("AsBool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValueAsArray(t *testing.T) {
	elems := []Value{NewInt(1), NewString("two")}

	got, err := NewArray(elems).AsArray()
	if err != nil {
		t.Fatalf("AsArray() unexpected error: %v", err)
	}
	if diff := cmp.Diff(elems, got, cmp.AllowUnexported(Value{})); diff != "" {
		t.Errorf("AsArray() mismatch (-want +got):\n%s", diff)
	}

	if _, err := NewString("x").AsArray(); !errors.Is(err, ErrKind) {
		t.Errorf("AsArray() on wrong kind error = %v, want error wrapping ErrKind", err)
	}
}

func TestValueAsStruct(t *testing.T) {
	members := map[string]Value{"a": NewInt(1)}

	got, err := NewStruct(members).AsStruct()
	if err != nil {
		t.Fatalf("AsStruct() unexpected error: %v", err)
	}
	if diff := cmp.Diff(members, got, cmp.AllowUnexported(Value{})); diff != "" {
		t.Errorf("AsStruct() mismatch (-want +got):\n%s", diff)
	}

	if _, err := NewString("x").AsStruct(); !errors.Is(err, ErrKind) {
		t.Errorf("AsStruct() on wrong kind error = %v, want error wrapping ErrKind", err)
	}
}

func TestValueAsBase64(t *testing.T) {
	tests := []struct {
		name    string
		value   Value
		want    []byte
		wantErr bool
	}{
		{name: "bytes", value: NewBase64([]byte{0x00, 0x01, 0xff}), want: []byte{0x00, 0x01, 0xff}},
		{name: "empty", value: NewBase64(nil), want: nil},
		{name: "wrong kind", value: NewString("foo"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.value.AsBase64()
			if tt.wantErr {
				if !errors.Is(err, ErrKind) {
					t.Errorf("AsBase64() error = %v, want error wrapping ErrKind", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("AsBase64() unexpected error: %v", err)
			}
			if !bytes.Equal(got, tt.want) {
				t.Errorf("AsBase64() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKindString(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindString, "string"},
		{KindInt, "int"},
		{KindInt64, "int64"},
		{KindDouble, "double"},
		{KindBool, "bool"},
		{KindArray, "array"},
		{KindStruct, "struct"},
		{KindNil, "nil"},
		{KindBase64, "base64"},
		{Kind(99), "Kind(99)"},
	}

	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("Kind(%d).String() = %q, want %q", int(tt.kind), got, tt.want)
		}
	}
}
