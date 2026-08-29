//go:build integration

package rtorrent_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/salimnassim/rtorrent"
)

func startRtorrent(t *testing.T) string {
	t.Helper()

	ctx := context.Background()

	c, err := testcontainers.Run(ctx, "",
		testcontainers.WithDockerfile(testcontainers.FromDockerfile{
			Context:    "testdata/rtorrent-integration",
			Dockerfile: "Dockerfile",
			Repo:       "rtorrent-integration",
			Tag:        "0.16.21",
			KeepImage:  true,
		}),
		testcontainers.WithFiles(testcontainers.ContainerFile{
			HostFilePath:      "testdata/fake.torrent",
			ContainerFilePath: "/downloads/fake.torrent",
			FileMode:          0o644,
		}),
		testcontainers.WithExposedPorts("5000/tcp"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("5000/tcp")),
	)
	testcontainers.CleanupContainer(t, c)
	if err != nil {
		t.Fatalf("start rtorrent container: %v", err)
	}

	addr, err := c.Endpoint(ctx, "")
	if err != nil {
		t.Fatalf("container endpoint: %v", err)
	}
	return addr
}

func TestIntegration_ListMethods(t *testing.T) {
	addr := startRtorrent(t)
	client := rtorrent.Dial(addr)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	v, err := client.Call(ctx, "system.listMethods")
	if err != nil {
		t.Fatalf("Call(system.listMethods) error: %v", err)
	}

	methods, err := v.AsArray()
	if err != nil {
		t.Fatalf("AsArray() error: %v", err)
	}
	if len(methods) == 0 {
		t.Error("system.listMethods returned no methods, want at least one")
	}
}

func loadFake(t *testing.T, client *rtorrent.Client) *rtorrent.Torrent {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := client.Call(ctx, "load.normal",
		rtorrent.NewString(""), rtorrent.NewString("/downloads/fake.torrent")); err != nil {
		t.Fatalf("Call(load.normal) error: %v", err)
	}

	return waitForFake(t, client)
}

func loadFakeRaw(t *testing.T, client *rtorrent.Client) *rtorrent.Torrent {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data, err := os.ReadFile("testdata/fake.torrent")
	if err != nil {
		t.Fatalf("ReadFile(testdata/fake.torrent) error: %v", err)
	}
	if err := client.LoadRaw(ctx, data); err != nil {
		t.Fatalf("LoadRaw() error: %v", err)
	}

	return waitForFake(t, client)
}

func waitForFake(t *testing.T, client *rtorrent.Client) *rtorrent.Torrent {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		torrents, err := client.Torrents(ctx, "main")
		if err != nil {
			t.Fatalf("Torrents() error: %v", err)
		}
		if len(torrents) > 0 {
			return torrents[0]
		}
		time.Sleep(250 * time.Millisecond)
	}

	t.Fatal("timed out waiting for fake.torrent to appear in the \"main\" view")
	return nil
}

func TestIntegration_LoadAndTorrents(t *testing.T) {
	addr := startRtorrent(t)
	client := rtorrent.Dial(addr)

	got := loadFake(t, client)

	if len(got.Hash) != 40 {
		t.Errorf("Torrent.Hash = %q, want 40 hex characters", got.Hash)
	}
	if got.Name != "fixture.txt" {
		t.Errorf("Torrent.Name = %q, want %q", got.Name, "fixture.txt")
	}
	if got.SizeBytes != 16 {
		t.Errorf("Torrent.SizeBytes = %d, want 16", got.SizeBytes)
	}
}

func TestIntegration_LoadRawAndTorrents(t *testing.T) {
	addr := startRtorrent(t)
	client := rtorrent.Dial(addr)

	got := loadFakeRaw(t, client)

	if len(got.Hash) != 40 {
		t.Errorf("Torrent.Hash = %q, want 40 hex characters", got.Hash)
	}
	if got.Name != "fixture.txt" {
		t.Errorf("Torrent.Name = %q, want %q", got.Name, "fixture.txt")
	}
	if got.SizeBytes != 16 {
		t.Errorf("Torrent.SizeBytes = %d, want 16", got.SizeBytes)
	}
}

func TestIntegration_FileMulticall(t *testing.T) {
	addr := startRtorrent(t)
	client := rtorrent.Dial(addr)

	torrent := loadFake(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	files, err := client.Files(ctx, torrent.Hash)
	if err != nil {
		t.Fatalf("Files(%q) error: %v", torrent.Hash, err)
	}
	if len(files) != 1 {
		t.Fatalf("Files(%q) returned %d files, want 1", torrent.Hash, len(files))
	}
	if files[0].Path != "fixture.txt" {
		t.Errorf("File.Path = %q, want %q", files[0].Path, "fixture.txt")
	}
	if files[0].SizeBytes != 16 {
		t.Errorf("File.SizeBytes = %d, want 16", files[0].SizeBytes)
	}
}

func TestIntegration_FaultUnwraps(t *testing.T) {
	addr := startRtorrent(t)
	client := rtorrent.Dial(addr)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err := client.Call(ctx, "not.a.real.method")
	if err == nil {
		t.Fatal("Call(not.a.real.method) error = nil, want a fault")
	}

	var fault *rtorrent.Fault
	if !errors.As(err, &fault) {
		t.Fatalf("errors.As(%v, *Fault) = false, want true", err)
	}
	if fault.FaultString == "" {
		t.Error("Fault.FaultString is empty, want a message from the server")
	}
}
