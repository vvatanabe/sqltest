package sqltest_test

import (
	"testing"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/vvatanabe/sqltest"
)

// TestDefaultMemcached demonstrates using NewMemcached with default options.
func TestDefaultMemcached(t *testing.T) {
	// Start a Memcached container with default options.
	client, cleanup := sqltest.NewMemcached(t)
	defer cleanup()

	// Initial data to store in Memcached.
	initialData := map[string]string{
		"user:1": `{"id": 1, "name": "Alice", "email": "alice@example.com"}`,
		"user:2": `{"id": 2, "name": "Bob", "email": "bob@example.com"}`,
	}

	// Prepare Memcached by storing initial data.
	if err := sqltest.PrepMemcached(t, client, initialData); err != nil {
		t.Fatalf("PrepMemcached failed: %v", err)
	}

	// Validate that the data was stored correctly.
	item, err := client.Get("user:1")
	if err != nil {
		t.Fatalf("failed to retrieve data: %v", err)
	}

	expectedValue := `{"id": 1, "name": "Alice", "email": "alice@example.com"}`
	if string(item.Value) != expectedValue {
		t.Errorf("expected value '%s', but got '%s'", expectedValue, string(item.Value))
	}
}

// TestMemcachedWithCustomRunOptions demonstrates overriding default RunOptions.
func TestMemcachedWithCustomRunOptions(t *testing.T) {
	// Custom RunOption to override the default tag.
	customTag := func(opts *dockertest.RunOptions) {
		opts.Tag = "1.5"
	}

	// Start a Memcached container with a custom tag.
	client, cleanup := sqltest.NewMemcachedWithOptions(t, []sqltest.RunOption{customTag})
	defer cleanup()

	// Test setting and getting a value.
	key := "product:123"
	value := `{"id": 123, "name": "Widget", "price": 19.99}`

	err := client.Set(&memcache.Item{
		Key:   key,
		Value: []byte(value),
	})
	if err != nil {
		t.Fatalf("failed to set value: %v", err)
	}

	// Validate the stored data.
	item, err := client.Get(key)
	if err != nil {
		t.Fatalf("failed to retrieve data: %v", err)
	}

	if string(item.Value) != value {
		t.Errorf("expected value '%s', but got '%s'", value, string(item.Value))
	}
}

// TestMemcachedWithCustomHostOptions demonstrates providing host configuration options.
func TestMemcachedWithCustomHostOptions(t *testing.T) {
	// Host option to set AutoRemove to true.
	autoRemove := func(hc *docker.HostConfig) {
		hc.AutoRemove = true
	}

	// Start a Memcached container with the AutoRemove option.
	client, cleanup := sqltest.NewMemcachedWithOptions(t, nil, autoRemove)
	defer cleanup()

	// Test multiple operations.
	// 1. Set a value
	key1 := "session:abc123"
	value1 := "user_id=456&expires=2023-12-31"
	err := client.Set(&memcache.Item{
		Key:   key1,
		Value: []byte(value1),
	})
	if err != nil {
		t.Fatalf("failed to set value: %v", err)
	}

	// 2. Set another value with expiration
	key2 := "temp:xyz789"
	value2 := "temporary data"
	err = client.Set(&memcache.Item{
		Key:        key2,
		Value:      []byte(value2),
		Expiration: 60, // 60 seconds
	})
	if err != nil {
		t.Fatalf("failed to set value with expiration: %v", err)
	}

	// 3. Validate the first value
	item, err := client.Get(key1)
	if err != nil {
		t.Fatalf("failed to retrieve data: %v", err)
	}
	if string(item.Value) != value1 {
		t.Errorf("expected value '%s', but got '%s'", value1, string(item.Value))
	}

	// 4. Validate the second value
	item, err = client.Get(key2)
	if err != nil {
		t.Fatalf("failed to retrieve data: %v", err)
	}
	if string(item.Value) != value2 {
		t.Errorf("expected value '%s', but got '%s'", value2, string(item.Value))
	}
}
