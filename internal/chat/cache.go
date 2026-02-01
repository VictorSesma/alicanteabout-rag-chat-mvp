package chat

import (
	"container/list"
	"sync"
)

type embedCache struct {
	mu    sync.Mutex
	max   int
	ll    *list.List
	items map[string]*list.Element
}

type embedEntry struct {
	key string
	vec []float32
}

func newEmbedCache(max int) *embedCache {
	return &embedCache{
		max:   max,
		ll:    list.New(),
		items: make(map[string]*list.Element),
	}
}

func (c *embedCache) Get(key string) ([]float32, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		ent := el.Value.(*embedEntry)
		return cloneVec(ent.vec), true
	}
	return nil, false
}

func (c *embedCache) Add(key string, vec []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		el.Value.(*embedEntry).vec = cloneVec(vec)
		return
	}
	ent := &embedEntry{key: key, vec: cloneVec(vec)}
	el := c.ll.PushFront(ent)
	c.items[key] = el
	if c.ll.Len() > c.max {
		old := c.ll.Back()
		if old != nil {
			c.ll.Remove(old)
			delete(c.items, old.Value.(*embedEntry).key)
		}
	}
}

func cloneVec(v []float32) []float32 {
	out := make([]float32, len(v))
	copy(out, v)
	return out
}
