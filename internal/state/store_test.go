package state

import (
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	store := New(filepath.Join(t.TempDir(), "nested", "config.json"))
	data, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if data.Settings.ListenPort != 1091 {
		t.Fatalf("default listen port = %d", data.Settings.ListenPort)
	}
	data.Subscriptions = append(data.Subscriptions, Subscription{ID: NewID(), Name: "example", URL: "https://example.com/sub"})
	if err := store.Save(data); err != nil {
		t.Fatal(err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Subscriptions) != 1 || loaded.Subscriptions[0].Name != "example" {
		t.Fatalf("unexpected subscriptions: %#v", loaded.Subscriptions)
	}
}
