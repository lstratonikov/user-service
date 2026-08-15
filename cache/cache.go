package cache

import (
	"sync"
	"time"
)

// Entry — запись в кэше с временем истечения
type Entry struct {
	Value     string
	ExpiresAt time.Time
}

// Cache — потокобезопасный кэш в памяти
// sync.Map безопасен для конкурентного доступа из горутин
type Cache struct {
	data sync.Map
	ttl  time.Duration // время жизни записи
}

func NewCache(ttl time.Duration) *Cache {
	c := &Cache{ttl: ttl}
	// запускаем горутину которая чистит устаревшие записи каждую минуту
	go c.cleanup()
	return c
}

// Set — сохраняет значение в кэше
func (c *Cache) Set(key string, value string) {
	c.data.Store(key, Entry{
		Value:     value,
		ExpiresAt: time.Now().Add(c.ttl),
	})
}

// Get — возвращает значение из кэша
// второй параметр false если запись не найдена или устарела
func (c *Cache) Get(key string) (string, bool) {
	val, ok := c.data.Load(key)
	if !ok {
		return "", false
	}

	entry := val.(Entry)

	// проверяем не истекло ли время жизни
	if time.Now().After(entry.ExpiresAt) {
		c.data.Delete(key)
		return "", false
	}

	return entry.Value, true
}

// Delete — удаляет запись из кэша
func (c *Cache) Delete(key string) {
	c.data.Delete(key)
}

// cleanup — периодически чистит устаревшие записи
// без этого кэш будет бесконечно расти в памяти
func (c *Cache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.data.Range(func(key, value any) bool {
			entry := value.(Entry)
			if time.Now().After(entry.ExpiresAt) {
				c.data.Delete(key)
			}
			return true
		})
	}
}
