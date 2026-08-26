// Command rtctl is a minimal CLI for calling XML-RPC methods.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/salimnassim/rtorrent"
)

func main() {
	err := run(os.Args[1:], os.Stdout)
	switch {
	case err == nil:
		return
	case errors.Is(err, flag.ErrHelp):
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stderr, "rtctl:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("rtctl", flag.ContinueOnError)
	addr := fs.String("addr", "", "rtorrent address: host:port or scgi://host:port (TCP SCGI), unix:///path (Unix socket SCGI), or http(s)://... (HTTP)")
	user := fs.String("user", "", "HTTP Basic Auth username (only used with an http(s):// -addr)")
	password := fs.String("password", "", "HTTP Basic Auth password (only used with an http(s):// -addr)")
	timeout := fs.Duration("timeout", rtorrent.DefaultTimeout, "request timeout")
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "usage: rtctl -addr ADDR [options] METHOD [PARAM ...]\n\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *addr == "" {
		return errors.New("-addr is required")
	}
	if fs.NArg() < 1 {
		return errors.New("method name is required")
	}
	method := fs.Arg(0)
	params := fs.Args()[1:]

	client, err := dial(*addr, *user, *password, *timeout)
	if err != nil {
		return err
	}

	values := make([]rtorrent.Value, len(params))
	for i, p := range params {
		values[i] = rtorrent.NewString(p)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	result, err := client.Call(ctx, method, values...)
	if err != nil {
		return err
	}

	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(toAny(result))
}

// dial builds a Client for addr, dispatching on its scheme: "scgi" or no
// scheme for TCP SCGI, "unix" for a Unix socket SCGI listener, and
// "http"/"https" for the HTTP transport (with Basic Auth if user is set).
func dial(addr, user, password string, timeout time.Duration) (*rtorrent.Client, error) {
	opts := []rtorrent.Option{rtorrent.WithTimeout(timeout)}

	scheme, rest, hasScheme := strings.Cut(addr, "://")
	if !hasScheme {
		return rtorrent.Dial(addr, opts...), nil
	}

	switch scheme {
	case "scgi":
		return rtorrent.Dial(rest, opts...), nil
	case "unix":
		return rtorrent.DialUnix(rest, opts...), nil
	case "http", "https":
		if user != "" {
			opts = append(opts, rtorrent.WithBasicAuth(user, password))
		}
		return rtorrent.DialHTTP(addr, opts...), nil
	default:
		return nil, fmt.Errorf("unsupported address scheme %q", scheme)
	}
}

// toAny converts an rtorrent.Value into a plain any tree suitable for
// json.Marshal.
func toAny(v rtorrent.Value) any {
	switch v.Kind() {
	case rtorrent.KindString:
		s, _ := v.AsString()
		return s
	case rtorrent.KindInt, rtorrent.KindInt64:
		n, _ := v.AsInt64()
		return n
	case rtorrent.KindDouble:
		d, _ := v.AsDouble()
		return d
	case rtorrent.KindBool:
		b, _ := v.AsBool()
		return b
	case rtorrent.KindArray:
		arr, _ := v.AsArray()
		out := make([]any, len(arr))
		for i, e := range arr {
			out[i] = toAny(e)
		}
		return out
	case rtorrent.KindStruct:
		strct, _ := v.AsStruct()
		out := make(map[string]any, len(strct))
		for k, e := range strct {
			out[k] = toAny(e)
		}
		return out
	default:
		return nil
	}
}
