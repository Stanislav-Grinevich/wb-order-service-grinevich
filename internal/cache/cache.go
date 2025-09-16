// Package cache реализует потокобезопасный in-memory кэш.
// Используется для более быстрого доступа к данным: сначала проверяется кэш,
// а при промахе выполняется запрос в базу.
package cache

import (
	"sync"

	"github.com/Stanislav-Grinevich/wb-order-service-grinevich/internal/models"
)

// Cache хранит заказы в памяти.
type Cache struct {
	mu   sync.RWMutex
	data map[string]models.Order
}

// New создаёт новый пустой кэш.
func New() *Cache {
	return &Cache{data: make(map[string]models.Order)}
}

// Get возвращает заказ по ID из кэша.
func (c *Cache) Get(id string) (models.Order, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	o, ok := c.data[id]
	return o, ok
}

// Set добавляет или обновляет заказ в кэше по его OrderUID.
func (c *Cache) Set(o models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[o.OrderUID] = o
}

// Load загружает список заказов в кэш.
// Используется при прогреве кэша при старте сервиса.
func (c *Cache) Load(os []models.Order) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, o := range os {
		c.data[o.OrderUID] = o
	}
}
