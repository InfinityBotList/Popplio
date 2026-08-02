package panel

import (
	"sync"
	"time"
)

// chunkTTL is how long an uploaded CDN chunk survives.
//
// The upstream doc comment says chunks expire after 10 minutes but the code sets
// moka's time_to_live to one hour. The CODE is authoritative.
const chunkTTL = 3600 * time.Second

// chunkCache is the in-memory store of uploaded CDN file chunks, keyed by a
// random 32-character id.
//
// It replaces moka's TTL cache: a mutex-guarded map plus a janitor goroutine.
// Take() must be atomic ("remove and return") because AddFile consumes each
// chunk exactly once while assembling a file.
type chunkCache struct {
	mu      sync.Mutex
	entries map[string]chunkEntry

	stop chan struct{}
	once sync.Once
}

type chunkEntry struct {
	data      []byte
	expiresAt time.Time
}

func newChunkCache() *chunkCache {
	c := &chunkCache{
		entries: make(map[string]chunkEntry),
		stop:    make(chan struct{}),
	}

	go c.janitor()

	return c
}

// Has reports whether a live chunk exists under the given id.
func (c *chunkCache) Has(id string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[id]

	return ok && time.Now().Before(entry.expiresAt)
}

// Put stores a chunk, replacing any existing entry under the same id.
func (c *chunkCache) Put(id string, data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[id] = chunkEntry{data: data, expiresAt: time.Now().Add(chunkTTL)}
}

// Take atomically removes a chunk and returns it.
func (c *chunkCache) Take(id string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[id]

	if !ok {
		return nil, false
	}

	delete(c.entries, id)

	if time.Now().After(entry.expiresAt) {
		return nil, false
	}

	return entry.data, true
}

// Close stops the janitor. Safe to call more than once.
func (c *chunkCache) Close() {
	c.once.Do(func() { close(c.stop) })
}

func (c *chunkCache) janitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.stop:
			return
		case now := <-ticker.C:
			c.mu.Lock()

			for id, entry := range c.entries {
				if now.After(entry.expiresAt) {
					delete(c.entries, id)
				}
			}

			c.mu.Unlock()
		}
	}
}
