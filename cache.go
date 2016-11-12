package main

import (
	"crypto/md5"
	"fmt"
	"sync"
	"time"

	"github.com/miekg/dns"
)

// KeyNotFound type
type KeyNotFound struct {
	key string
}

// Error formats an error for the KeyNotFound type
func (e KeyNotFound) Error() string {
	return e.key + " " + "not found"
}

// KeyExpired type
type KeyExpired struct {
	Key string
}

// Error formats an error for the KeyExpired type
func (e KeyExpired) Error() string {
	return e.Key + " " + "expired"
}

// CacheIsFull type
type CacheIsFull struct {
}

// Error formats an error for the CacheIsFull type
func (e CacheIsFull) Error() string {
	return "Cache is Full"
}

// SerializerError type
type SerializerError struct {
}

// Error formats an error for the SerializerError type
func (e SerializerError) Error() string {
	return "Serializer error"
}

// Mesg represents a cache entry
type Mesg struct {
	Msg     *dns.Msg
	Blocked bool
	Expire  time.Time
}

// Cache interface
type Cache interface {
	Get(key string) (Msg *dns.Msg, blocked bool, err error)
	Set(key string, Msg *dns.Msg, blocked bool) error
	Exists(key string) bool
	Remove(key string)
	Length() int
}

// MemoryCache type
type MemoryCache struct {
	Backend  map[string]Mesg
	Expire   time.Duration
	Maxcount int
	mu       sync.RWMutex
}

// MemoryBlockCache type
type MemoryBlockCache struct {
	Backend map[string]bool
	mu      sync.RWMutex
}

// MemoryQuestionCache type
type MemoryQuestionCache struct {
	Backend  []QuestionCacheEntry `json:"entry"`
	Maxcount int
	mu       sync.RWMutex
}

// Get returns the entry for a key or an error
func (c *MemoryCache) Get(key string) (*dns.Msg, bool, error) {
	c.mu.RLock()
	mesg, ok := c.Backend[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false, KeyNotFound{key}
	}

	if mesg.Expire.Before(time.Now()) {
		c.Remove(key)
		return nil, false, KeyExpired{key}
	}

	return mesg.Msg, mesg.Blocked, nil
}

// Set sets a keys value to a Mesg
func (c *MemoryCache) Set(key string, msg *dns.Msg, blocked bool) error {
	if c.Full() && !c.Exists(key) {
		return CacheIsFull{}
	}

	expire := time.Now().Add(c.Expire)
	mesg := Mesg{msg, blocked, expire}
	c.mu.Lock()
	c.Backend[key] = mesg
	c.mu.Unlock()

	return nil
}

// Remove removes an entry from the cache
func (c *MemoryCache) Remove(key string) {
	c.mu.Lock()
	delete(c.Backend, key)
	c.mu.Unlock()
}

// Exists returns whether or not a key exists in the cache
func (c *MemoryCache) Exists(key string) bool {
	c.mu.RLock()
	_, ok := c.Backend[key]
	c.mu.RUnlock()
	return ok
}

// Length returns the caches length
func (c *MemoryCache) Length() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.Backend)
}

// Full returns whether or not the cache is full
func (c *MemoryCache) Full() bool {
	if c.Maxcount == 0 {
		return false
	}
	return c.Length() >= c.Maxcount
}

// KeyGen generates a key for the hash from a question
func KeyGen(q Question) string {
	h := md5.New()
	h.Write([]byte(q.String()))
	x := h.Sum(nil)
	key := fmt.Sprintf("%x", x)
	return key
}

// Get returns the entry for a key or an error
func (c *MemoryBlockCache) Get(key string) (bool, error) {
	c.mu.RLock()
	val, ok := c.Backend[key]
	c.mu.RUnlock()

	if !ok {
		return false, KeyNotFound{key}
	}

	return val, nil
}

// Remove removes an entry from the cache
func (c *MemoryBlockCache) Remove(key string) {
	c.mu.Lock()
	delete(c.Backend, key)
	c.mu.Unlock()
}

// Set sets a value in the BlockCache
func (c *MemoryBlockCache) Set(key string, value bool) error {
	c.mu.Lock()
	c.Backend[key] = value
	c.mu.Unlock()

	return nil
}

// Exists returns whether or not a key exists in the cache
func (c *MemoryBlockCache) Exists(key string) bool {
	c.mu.RLock()
	_, ok := c.Backend[key]
	c.mu.RUnlock()
	return ok
}

// Length returns the caches length
func (c *MemoryBlockCache) Length() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.Backend)
}

// Add adds a question to the cache
func (c *MemoryQuestionCache) Add(q QuestionCacheEntry) {
	c.mu.Lock()
	if c.Maxcount != 0 && len(c.Backend) >= c.Maxcount {
		c.Backend = nil
	}
	c.Backend = append(c.Backend, q)
	c.mu.Unlock()
}

// Clear clears the contents of the cache
func (c *MemoryQuestionCache) Clear() {
	c.mu.Lock()
	c.Backend = nil
	c.mu.Unlock()
}

// Length returns the caches length
func (c *MemoryQuestionCache) Length() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.Backend)
}
