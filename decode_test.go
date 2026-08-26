package rtorrent

import (
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDecodeMethodResponse(t *testing.T) {
	tests := []struct {
		name    string
		xml     string
		want    Value
		wantErr bool
	}{
		{
			name: "string",
			xml:  `<methodResponse><params><param><value><string>hello</string></value></param></params></methodResponse>`,
			want: NewString("hello"),
		},
		{
			name: "i4",
			xml:  `<methodResponse><params><param><value><i4>42</i4></value></param></params></methodResponse>`,
			want: NewInt(42),
		},
		{
			name: "int alias for i4",
			xml:  `<methodResponse><params><param><value><int>7</int></value></param></params></methodResponse>`,
			want: NewInt(7),
		},
		{
			name: "i8",
			xml:  `<methodResponse><params><param><value><i8>9223372036854775807</i8></value></param></params></methodResponse>`,
			want: NewInt64(9223372036854775807),
		},
		{
			name: "double",
			xml:  `<methodResponse><params><param><value><double>3.25</double></value></param></params></methodResponse>`,
			want: NewDouble(3.25),
		},
		{
			name: "boolean true",
			xml:  `<methodResponse><params><param><value><boolean>1</boolean></value></param></params></methodResponse>`,
			want: NewBool(true),
		},
		{
			name: "boolean false",
			xml:  `<methodResponse><params><param><value><boolean>0</boolean></value></param></params></methodResponse>`,
			want: NewBool(false),
		},
		{
			name: "self-closing nil",
			xml:  `<methodResponse><params><param><value><nil/></value></param></params></methodResponse>`,
			want: NewNil(),
		},
		{
			name: "nested array",
			xml: `<methodResponse><params><param><value><array><data>` +
				`<value><i4>1</i4></value><value><i4>2</i4></value>` +
				`</data></array></value></param></params></methodResponse>`,
			want: NewArray([]Value{NewInt(1), NewInt(2)}),
		},
		{
			name: "nested struct",
			xml: `<methodResponse><params><param><value><struct>` +
				`<member><name>a</name><value><i4>1</i4></value></member>` +
				`<member><name>b</name><value><string>two</string></value></member>` +
				`</struct></value></param></params></methodResponse>`,
			want: NewStruct(map[string]Value{"a": NewInt(1), "b": NewString("two")}),
		},
		{
			name: "self-closing name in struct member",
			xml: `<methodResponse><params><param><value><struct>` +
				`<member><name/><value><i4>5</i4></value></member>` +
				`</struct></value></param></params></methodResponse>`,
			want: NewStruct(map[string]Value{"": NewInt(5)}),
		},
		{
			name: "unrecognized scalar tag degrades to string",
			xml: `<methodResponse><params><param><value>` +
				`<dateTime.iso8601>20260826T00:00:00</dateTime.iso8601>` +
				`</value></param></params></methodResponse>`,
			want: NewString("20260826T00:00:00"),
		},
		{
			name: "unrecognized tag with children errors",
			xml: `<methodResponse><params><param><value>` +
				`<weird><nested>x</nested></weird>` +
				`</value></param></params></methodResponse>`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := decodeMethodResponse([]byte(tt.xml))
			if tt.wantErr {
				if err == nil {
					t.Fatalf("decodeMethodResponse() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("decodeMethodResponse() unexpected error: %v", err)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(Value{})); diff != "" {
				t.Errorf("decodeMethodResponse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDecodeMethodResponseFault(t *testing.T) {
	data := `<methodResponse><fault><value><struct>` +
		`<member><name>faultCode</name><value><i4>7</i4></value></member>` +
		`<member><name>faultString</name><value><string>method not found</string></value></member>` +
		`</struct></value></fault></methodResponse>`

	_, err := decodeMethodResponse([]byte(data))
	if err == nil {
		t.Fatalf("decodeMethodResponse() error = nil, want *Fault")
	}

	var fault *Fault
	if !errors.As(err, &fault) {
		t.Fatalf("errors.As(%v, &fault) = false, want true", err)
	}
	if fault.FaultCode != 7 {
		t.Errorf("fault.FaultCode = %d, want 7", fault.FaultCode)
	}
	if fault.FaultString != "method not found" {
		t.Errorf("fault.FaultString = %q, want %q", fault.FaultString, "method not found")
	}
}

func TestDecodeMethodResponseTruncated(t *testing.T) {
	_, err := decodeMethodResponse([]byte(`<methodResponse><params><param><value><string>oops`))
	if err == nil {
		t.Fatalf("decodeMethodResponse() error = nil, want error for truncated input")
	}
	if !strings.Contains(err.Error(), "decode xml-rpc") {
		t.Errorf("decodeMethodResponse() error = %v, want it to mention decode xml-rpc", err)
	}
}
