package rtorrent_test

import (
	"context"
	"fmt"
	"time"

	"github.com/salimnassim/rtorrent"
)

func Example_dial() {
	client := rtorrent.Dial("127.0.0.1:5000", rtorrent.WithTimeout(5*time.Second))

	ctx := context.Background()
	if _, err := client.Call(ctx, "system.listMethods"); err != nil {
		fmt.Println(err)
	}
}

func Example_torrents() {
	client := rtorrent.Dial("127.0.0.1:5000")

	ctx := context.Background()
	torrents, err := client.Torrents(ctx, "main")
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, t := range torrents {
		fmt.Println(t.Name, t.SizeBytes)
	}
}

func Example_custom() {
	client := rtorrent.Dial("127.0.0.1:5000")

	ctx := context.Background()
	rows, err := client.TorrentsCustom(ctx, "main", "d.hash=", "d.name=")
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, row := range rows {
		hash, _ := row[0].AsString()
		name, _ := row[1].AsString()
		fmt.Println(hash, name)
	}
}
