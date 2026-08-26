# rtorrent

A dependency-free client for rTorrent's XML-RPC interface. Communicates
directly over the SCGI listener (TCP or Unix socket). No proxy
required for a backend/native client, has a HTTP transport as a fallback
for proxy setups.

## Contents

- `value.go` `Kind`, `Value`, constructors, accessors.
- `encode.go` `encodeMethodCall`, `encodeValue`, XML escaping.
- `decode.go` `decodeMethodResponse`, `rpcParser`
- `fault.go` `Fault` type, `faultFromValue`.
- `transport.go` `transport` interface.
- `scgi.go` `scgiTransport`: TCP and Unix socket, direct to rTorrent.
- `http.go` `httpTransport`: for proxy setups.
- `client.go` `Client`, `Dial`/`DialUnix`/`DialHTTP`, `Call`, `Multicall`,
  `multicallRows`.
- `torrent.go` `Torrent` model, `Client.Torrents`, `Client.TorrentsCustom`.
- `peer.go` `Peer` model, `Client.Peers`, `Client.PeersCustom`.
- `tracker.go` `Tracker` model, `Client.Trackers`, `Client.TrackersCustom`.
- `file.go` `File` model, `Client.Files`, `Client.FilesCustom`.

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
## Development

```
make test   # go test -v ./...
make test-integration   # go test -tags=integration -v ./...
make ci     # gofmt check, build, vet, race tests, mod tidy check, govulncheck
```