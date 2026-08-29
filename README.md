# rtorrent

A dependency-free client for rTorrent's XML-RPC interface. Talks directly to
rTorrent's SCGI listener (TCP or Unix socket), no proxy needed, with an
HTTP transport as a fallback for proxied setups.

## Contents

- `value.go` `Kind`, `Value`, constructors, accessors.
- `encode.go` `encodeMethodCall`, `encodeValue`, XML escaping.
- `decode.go` `decodeMethodResponse`, `rpcParser`
- `fault.go` `Fault` type, `faultFromValue`.
- `transport.go` `transport` interface.
- `scgi.go` `scgiTransport`: TCP and Unix socket, direct to rTorrent.
- `http.go` `httpTransport`: for proxy setups.
- `client.go` `Client`, `Dial`/`DialUnix`/`DialHTTP`, `Call`, `Multicall`,
  `multicallRows`, `LoadRaw`/`LoadRawStart`/`LoadStart`,
  `SetCustom1`-`SetCustom5`, `Start`/`Stop`/`Pause`/`Resume`/
  `OpenTorrent`/`CloseTorrent`/`Erase`, `SetPriority`/`SetFilePriority`,
  `SetDirectory`/`CheckHash`, `Option`s (`WithTimeout`, `WithBasicAuth`).
- `torrent.go` `Torrent` model, `Client.Torrents`, `Client.TorrentsCustom`.
- `peer.go` `Peer` model, `Client.Peers`, `Client.PeersCustom`.
- `tracker.go` `Tracker` model, `Client.Trackers`, `Client.TrackersCustom`.
- `file.go` `File` model, `Client.Files`, `Client.FilesCustom`.
- `execute.go` `Client.ExecuteThrow`/`ExecuteNothrow`/`ExecuteCapture`.
- `cmd/rtctl.go` `rtctl`, a minimal CLI for calling XML-RPC methods directly.

## Install

```
go get github.com/salimnassim/rtorrent
```

## Usage

```go
client := rtorrent.Dial("127.0.0.1:5000")

ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

torrents, err := client.Torrents(ctx, "main")
if err != nil {
	log.Fatal(err)
}
for _, t := range torrents {
	fmt.Println(t.Name, t.SizeBytes)
}
```

For an HTTP-proxied setup behind Basic Auth:

```go
client := rtorrent.DialHTTP("https://example.com/RPC2",
	rtorrent.WithBasicAuth("user", "pass"))
```

### Multicall (d.multicall2)

`Client.Multicall` drives any `d.*`/`t.*`/`p.*`/`f.*` multicall method
directly, for cases `Torrents`/`Peers`/`Trackers`/`Files` don't cover:

```go
rows, err := client.Multicall(ctx, "d.multicall2",
	[]rtorrent.Value{rtorrent.NewString(""), rtorrent.NewString("main")},
	"d.hash=", "d.name=")
if err != nil {
	log.Fatal(err)
}
for _, row := range rows {
	hash, _ := row[0].AsString()
	name, _ := row[1].AsString()
	fmt.Println(hash, name)
}
```

### Batching calls (system.multicall)

`system.multicall` batches unrelated method calls into a single round trip.
Its request/response shape differs from `d.multicall2` (a struct per call
rather than one row per torrent), so it's driven through `Call` directly:

```go
calls := rtorrent.NewArray([]rtorrent.Value{
	rtorrent.NewStruct(map[string]rtorrent.Value{
		"methodName": rtorrent.NewString("system.listMethods"),
		"params":     rtorrent.NewArray(nil),
	}),
	rtorrent.NewStruct(map[string]rtorrent.Value{
		"methodName": rtorrent.NewString("system.client_version"),
		"params":     rtorrent.NewArray(nil),
	}),
})

v, err := client.Call(ctx, "system.multicall", calls)
if err != nil {
	log.Fatal(err)
}
results, _ := v.AsArray()
for _, r := range results {
	fmt.Println(r)
}
```

### Torrent control

Lifecycle, priority, and directory/rehash actions are plain `Client`
methods, each a thin wrapper over one `d.*`/`f.*` command:

```go
if err := client.Stop(ctx, hash); err != nil {
	log.Fatal(err)
}
if err := client.SetPriority(ctx, hash, 3); err != nil { // 3 = high
	log.Fatal(err)
}
if err := client.CheckHash(ctx, hash); err != nil {
	log.Fatal(err)
}
```

### Running commands (execute)

`ExecuteThrow` runs an external command, returning an error
(a `*Fault`) if it exits non-zero. `ExecuteNothrow` never
faults on a failing command, it returns the exit status directly.
`ExecuteCapture` runs a command and returns its captured stdout, faulting
on a non-zero exit.

```go
out, err := client.ExecuteCapture(ctx, "echo", "-n", "hello")
if err != nil {
	log.Fatal(err)
}
fmt.Println(out) // "hello"

status, err := client.ExecuteNothrow(ctx, "some-script.sh")
if err != nil {
	log.Fatal(err) // only a transport/decode error, not a nonzero exit
}
fmt.Println("exit status:", status)
```

### Loading a torrent from raw bytes (load.raw)

`LoadRaw` loads a torrent from its bencoded bytes directly, without writing
a `.torrent` file to disk first:

```go
data, err := os.ReadFile("example.torrent")
if err != nil {
	log.Fatal(err)
}
if err := client.LoadRaw(ctx, data); err != nil {
	log.Fatal(err)
}
```

Use `LoadRawStart` instead to start the torrent immediately after loading.

## CLI

`rtctl` is a minimal CLI for calling XML-RPC methods directly:

```
make build
rtctl -addr 127.0.0.1:5000 system.listMethods
rtctl -addr https://foo.bar.tld/RPC2 -user bob -password secret d.multicall2 "" main d.hash= d.name=
```

Or run from source without building:

```
go run ./cmd -addr 127.0.0.1:5000 system.listMethods
go run ./cmd -addr https://foo.bar.tld/RPC2 -user bob -password secret d.multicall2 "" main d.hash= d.name=
```

## Development

```
make test              # go test -v ./...
make test-integration  # go test -tags=integration -v ./...
make fuzz              # fuzz FuzzDecodeMethodResponse, FuzzReadSCGIResponse, FuzzEncodeDecodeString, FuzzEncodeDecodeBase64, 30s each
make ci                # gofmt check, build, vet, race tests, mod tidy check, govulncheck
```

`make test-integration` launches a real rTorrent 0.16.21 in Docker
(testcontainers-go) and needs Docker + network access, so it's kept out of
`make test` and the default CI workflow.

## License

MIT, see [LICENSE](LICENSE).