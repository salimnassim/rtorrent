package rtorrent

import (
	"context"
	"fmt"
)

// execute invokes an execute-family RPC method with command and args.
func (c *Client) execute(ctx context.Context, name, command string, args []string) (Value, error) {
	params := make([]Value, 0, len(args)+2)
	params = append(params, NewString(""), NewString(command))
	for _, arg := range args {
		params = append(params, NewString(arg))
	}
	return c.Call(ctx, name, params...)
}

// ExecuteThrow runs command with args via execute.throw.
func (c *Client) ExecuteThrow(ctx context.Context, command string, args ...string) error {
	_, err := c.execute(ctx, "execute.throw", command, args)
	return err
}

// ExecuteNothrow runs command with args via execute.nothrow and returns the
// process's exit status.
func (c *Client) ExecuteNothrow(ctx context.Context, command string, args ...string) (int64, error) {
	v, err := c.execute(ctx, "execute.nothrow", command, args)
	if err != nil {
		return 0, err
	}
	status, err := v.AsInt64()
	if err != nil {
		return 0, fmt.Errorf("rtorrent: execute.nothrow: %w", err)
	}
	return status, nil
}

// ExecuteCapture runs command with args via execute.capture and returns its
// captured standard output.
func (c *Client) ExecuteCapture(ctx context.Context, command string, args ...string) (string, error) {
	v, err := c.execute(ctx, "execute.capture", command, args)
	if err != nil {
		return "", err
	}
	out, err := v.AsString()
	if err != nil {
		return "", fmt.Errorf("rtorrent: execute.capture: %w", err)
	}
	return out, nil
}
