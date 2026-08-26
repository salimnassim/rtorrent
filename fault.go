package rtorrent

import "fmt"

// Fault represents an XML-RPC fault.
type Fault struct {
	// FaultCode is the numeric fault code reported by the server.
	FaultCode int
	// FaultString is the fault message reported by the server.
	FaultString string
}

// Error implements the error interface for Fault.
func (f *Fault) Error() string {
	return fmt.Sprintf("xml-rpc fault %d: %s", f.FaultCode, f.FaultString)
}

// faultFromValue converts a decoded XML-RPC <fault> struct into a Fault.
func faultFromValue(v Value) (*Fault, error) {
	members, err := v.AsStruct()
	if err != nil {
		return nil, fmt.Errorf("decode xml-rpc fault: %w", err)
	}

	codeValue, ok := members["faultCode"]
	if !ok {
		return nil, fmt.Errorf("decode xml-rpc fault: missing faultCode member")
	}
	code, err := codeValue.AsInt64()
	if err != nil {
		return nil, fmt.Errorf("decode xml-rpc fault: faultCode: %w", err)
	}

	stringValue, ok := members["faultString"]
	if !ok {
		return nil, fmt.Errorf("decode xml-rpc fault: missing faultString member")
	}
	str, err := stringValue.AsString()
	if err != nil {
		return nil, fmt.Errorf("decode xml-rpc fault: faultString: %w", err)
	}

	return &Fault{FaultCode: int(code), FaultString: str}, nil
}
