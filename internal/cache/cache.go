package cache

import (
	"sync"

	"github.com/stani/wb-order-service-grinevich/internal/models"
)

type Cache struct {
	mu   sync.RWMutex
	data map[string]models.Order
}

func New() *Cache { return &Cache{data: make(map[string]models.Order)} }

func (c *Cache) Get(id string) (models.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	o, ok := c.data[id]
	return o, ok
}

func (c *Cache) Set(o models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[o.OrderUID] = o
}

func (c *Cache) Load(os []models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, o := range os {
		c.data[o.OrderUID] = o
	}
}
