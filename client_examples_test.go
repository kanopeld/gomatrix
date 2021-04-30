package gomatrix

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestClient_Sync(t *testing.T) {
	cli, _ := NewClient("https://matrix.org", "@example:matrix.org", "MDAefhiuwehfuiwe")
	cli.Store.SaveFilterID("@example:matrix.org", "2")                // Optional: if you know it already
	cli.Store.SaveNextBatch("@example:matrix.org", "111_222_333_444") // Optional: if you know it already
	cli.On(MessageEventType, func(ev *Event) {
		fmt.Println("Message: ", ev)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// Blocking version
	if err := cli.Sync(ctx); err != nil {
		fmt.Println("Sync() returned ", err)
	}

	// Non-blocking version
	go func() {
		for {
			if err := cli.Sync(ctx); err != nil {
				fmt.Println("Sync() returned ", err)
			}
			// Optional: Wait a period of time before trying to sync again.
		}
	}()
}
