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

func loadFakeStart(t *testing.T, client *rtorrent.Client) *rtorrent.Torrent {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.LoadStart(ctx, "/downloads/fake.torrent"); err != nil {
		t.Fatalf("LoadStart() error: %v", err)
	}

	return waitForFake(t, client)
}

func loadFakeRawStart(t *testing.T, client *rtorrent.Client) *rtorrent.Torrent {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	data, err := os.ReadFile("testdata/fake.torrent")
	if err != nil {
		t.Fatalf("ReadFile(testdata/fake.torrent) error: %v", err)
	}
	if err := client.LoadRawStart(ctx, data); err != nil {
		t.Fatalf("LoadRawStart() error: %v", err)
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

func mustTorrent(t *testing.T, client *rtorrent.Client, hash string) *rtorrent.Torrent {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	torrents, err := client.Torrents(ctx, "main")
	if err != nil {
		t.Fatalf("Torrents() error: %v", err)
	}
	for _, tr := range torrents {
		if tr.Hash == hash {
			return tr
		}
	}

	t.Fatalf("torrent %q not found in \"main\" view", hash)
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

func TestIntegration_LoadStartAndTorrents(t *testing.T) {
	addr := startRtorrent(t)
	client := rtorrent.Dial(addr)

	got := loadFakeStart(t, client)

	if len(got.Hash) != 40 {
		t.Errorf("Torrent.Hash = %q, want 40 hex characters", got.Hash)
	}
	if got.Name != "fixture.txt" {
		t.Errorf("Torrent.Name = %q, want %q", got.Name, "fixture.txt")
	}
	if !got.IsActive {
		t.Error("Torrent.IsActive = false, want true after load.start")
	}
}

func TestIntegration_LoadRawStartAndTorrents(t *testing.T) {
	addr := startRtorrent(t)
	client := rtorrent.Dial(addr)

	got := loadFakeRawStart(t, client)

	if len(got.Hash) != 40 {
		t.Errorf("Torrent.Hash = %q, want 40 hex characters", got.Hash)
	}
	if got.Name != "fixture.txt" {
		t.Errorf("Torrent.Name = %q, want %q", got.Name, "fixture.txt")
	}
	if !got.IsActive {
		t.Error("Torrent.IsActive = false, want true after load.raw_start")
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

func TestIntegration_TorrentLifecycle(t *testing.T) {
	addr := startRtorrent(t)
	client := rtorrent.Dial(addr)

	torrent := loadFake(t, client)
	hash := torrent.Hash

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.Start(ctx, hash); err != nil {
		t.Fatalf("Start(%q) error: %v", hash, err)
	}
	if got := mustTorrent(t, client, hash); !got.IsActive {
		t.Error("Torrent.IsActive = false after Start, want true")
	}

	if err := client.Stop(ctx, hash); err != nil {
		t.Fatalf("Stop(%q) error: %v", hash, err)
	}
	if got := mustTorrent(t, client, hash); got.IsActive {
		t.Error("Torrent.IsActive = true after Stop, want false")
	}

	if err := client.Pause(ctx, hash); err != nil {
		t.Fatalf("Pause(%q) error: %v", hash, err)
	}
	if err := client.Resume(ctx, hash); err != nil {
		t.Fatalf("Resume(%q) error: %v", hash, err)
	}

	if err := client.OpenTorrent(ctx, hash); err != nil {
		t.Fatalf("OpenTorrent(%q) error: %v", hash, err)
	}
	if err := client.CloseTorrent(ctx, hash); err != nil {
		t.Fatalf("CloseTorrent(%q) error: %v", hash, err)
	}

	if err := client.SetPriority(ctx, hash, 3); err != nil {
		t.Fatalf("SetPriority(%q, 3) error: %v", hash, err)
	}
	if got := mustTorrent(t, client, hash); got.Priority != 3 {
		t.Errorf("Torrent.Priority = %d after SetPriority(3), want 3", got.Priority)
	}

	if err := client.SetDirectory(ctx, hash, "/downloads"); err != nil {
		t.Fatalf("SetDirectory(%q, /downloads) error: %v", hash, err)
	}
	if got := mustTorrent(t, client, hash); got.Directory != "/downloads" {
		t.Errorf("Torrent.Directory = %q after SetDirectory, want %q", got.Directory, "/downloads")
	}

	if err := client.CheckHash(ctx, hash); err != nil {
		t.Fatalf("CheckHash(%q) error: %v", hash, err)
	}

	if err := client.Erase(ctx, hash); err != nil {
		t.Fatalf("Erase(%q) error: %v", hash, err)
	}
	torrents, err := client.Torrents(ctx, "main")
	if err != nil {
		t.Fatalf("Torrents() error: %v", err)
	}
	for _, tr := range torrents {
		if tr.Hash == hash {
			t.Errorf("torrent %q still present in \"main\" view after Erase", hash)
		}
	}
}

func TestIntegration_SetFilePriority(t *testing.T) {
	addr := startRtorrent(t)
	client := rtorrent.Dial(addr)

	torrent := loadFake(t, client)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := client.SetFilePriority(ctx, torrent.Hash, 0, 2); err != nil {
		t.Fatalf("SetFilePriority(%q, 0, 2) error: %v", torrent.Hash, err)
	}

	files, err := client.Files(ctx, torrent.Hash)
	if err != nil {
		t.Fatalf("Files(%q) error: %v", torrent.Hash, err)
	}
	if len(files) != 1 {
		t.Fatalf("Files(%q) returned %d files, want 1", torrent.Hash, len(files))
	}
	if files[0].Priority != 2 {
		t.Errorf("File.Priority = %d after SetFilePriority(2), want 2", files[0].Priority)
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
