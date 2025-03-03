package sqltest

import (
	"fmt"
	"testing"
	"time"

	"github.com/bradfitz/gomemcache/memcache"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	defaultMemcachedImage = "memcached"
	defaultMemcachedTag   = "1.6"
)

// NewMemcached starts a Memcached Docker container using the default settings and returns a connected *memcache.Client
// along with a cleanup function. It uses the default Memcached image ("memcached") with tag "1.6". For more
// customization, use NewMemcachedWithOptions.
func NewMemcached(t testing.TB) (*memcache.Client, func()) {
	return NewMemcachedWithOptions(t, nil)
}

// NewMemcachedWithOptions starts a Memcached Docker container using Docker and returns a connected *memcache.Client
// along with a cleanup function. It applies the default settings:
//   - Repository: "memcached"
//   - Tag: "1.6"
//
// Additional RunOption functions can be provided via the runOpts parameter to override these defaults,
// and optional host configuration functions can be provided via hostOpts.
func NewMemcachedWithOptions(t testing.TB, runOpts []RunOption, hostOpts ...func(*docker.HostConfig)) (*memcache.Client, func()) {
	t.Helper()

	// Set default run options for Memcached
	defaultRunOpts := &dockertest.RunOptions{
		Repository: defaultMemcachedImage,
		Tag:        defaultMemcachedTag,
	}

	// Apply any provided RunOption functions to override defaults
	for _, opt := range runOpts {
		opt(defaultRunOpts)
	}

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("failed to connect to docker: %s", err)
	}

	// Pass optional host configuration options
	resource, err := pool.RunWithOptions(defaultRunOpts, hostOpts...)
	if err != nil {
		t.Fatalf("failed to start memcached container: %s", err)
	}

	// Get the host port that was assigned to the container's 11211 port
	actualPort := resource.GetHostPort("11211/tcp")
	if actualPort == "" {
		_ = pool.Purge(resource)
		t.Fatalf("no host port was assigned for the memcached container")
	}
	t.Logf("memcached container is running on host port '%s'", actualPort)

	// Create a memcache client
	var client *memcache.Client
	if err = pool.Retry(func() error {
		client = memcache.New(actualPort)
		// Test the connection by setting and getting a value
		testKey := "test_connection"
		testValue := []byte("test_value")
		err := client.Set(&memcache.Item{
			Key:   testKey,
			Value: testValue,
		})
		if err != nil {
			return err
		}

		// Wait a moment to ensure the value is stored
		time.Sleep(100 * time.Millisecond)

		_, err = client.Get(testKey)
		return err
	}); err != nil {
		_ = pool.Purge(resource)
		t.Fatalf("failed to connect to memcached: %s", err)
	}

	cleanup := func() {
		if err := pool.Purge(resource); err != nil {
			t.Logf("failed to remove memcached container: %s", err)
		}
	}

	return client, cleanup
}

// PrepMemcached sets initial key-value pairs in the Memcached instance.
// It takes a map of key-value pairs and stores them in the cache.
func PrepMemcached(t testing.TB, client *memcache.Client, initialData map[string]string) error {
	t.Helper()

	for key, value := range initialData {
		err := client.Set(&memcache.Item{
			Key:   key,
			Value: []byte(value),
		})
		if err != nil {
			return fmt.Errorf("failed to set key '%s': %w", key, err)
		}
	}
	return nil
}
