package tui

import "sync"

// UserCache provides a thread-safe cache for mapping Slack user IDs to display names.
type UserCache struct {
	mu    sync.RWMutex
	users map[string]string
}

func NewUserCache() *UserCache {
	return &UserCache{users: make(map[string]string)}
}

func (c *UserCache) Get(id string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	name, ok := c.users[id]
	return name, ok
}

func (c *UserCache) Set(id, name string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.users[id] = name
}
