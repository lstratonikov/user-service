package loadtest

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"time"
	"user-service/metrics"
)

// Bullet — патрон, данные реального пользователя из БД
type Bullet struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	StatusID int    `json:"status_id"`
}

// BulletsCollector — собирает патроны из БД и сохраняет в файл
type BulletsCollector struct {
	db       *sql.DB
	filePath string
	needed   int
}

func NewBulletsCollector(db *sql.DB, filePath string) *BulletsCollector {
	return &BulletsCollector{
		db:       db,
		filePath: filePath,
		needed:   100000,
	}
}

// Start — запускает планировщик сбора патронов каждый день в 9:45
func (c *BulletsCollector) Start() {
	go func() {
		for {
			now := time.Now()

			next := time.Date(now.Year(), now.Month(), now.Day(), 9, 20, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}

			waitDuration := next.Sub(now)
			log.Printf("Bullets collector: next collection at %s (in %s)",
				next.Format("15:04:05"), waitDuration.Round(time.Second))

			time.Sleep(waitDuration)
			c.Collect()
		}
	}()
}

// Collect — собирает патроны из БД и сохраняет в JSON файл
func (c *BulletsCollector) Collect() {
	log.Println("Bullets collector: starting collection...")

	rows, err := c.db.Query(`
		SELECT id, email, phone, status_id
		FROM users 
		ORDER BY RANDOM() 
		LIMIT $1
	`, c.needed)
	if err != nil {
		log.Printf("Bullets collector: failed to query: %v", err)
		return
	}
	defer rows.Close()

	var bullets []Bullet
	for rows.Next() {
		var b Bullet
		var phone sql.NullString
		var statusID int
		if err := rows.Scan(&b.ID, &b.Email, &phone, &statusID); err != nil {
			continue
		}
		if phone.Valid {
			b.Phone = phone.String
		}
		b.StatusID = statusID
		bullets = append(bullets, b)
	}

	// проверяем ошибку после итерации
	if err := rows.Err(); err != nil {
		log.Printf("Bullets collector: rows iteration error: %v", err)
		return
	}

	if len(bullets) < c.needed {
		log.Printf("Bullets collector: WARNING only %d bullets collected, needed %d",
			len(bullets), c.needed)
	}

	data, err := json.Marshal(bullets)
	if err != nil {
		log.Printf("Bullets collector: failed to marshal: %v", err)
		return
	}

	if err := os.WriteFile(c.filePath, data, 0644); err != nil {
		log.Printf("Bullets collector: failed to write file: %v", err)
		return
	}

	// обновляем метрики после сборки
	metrics.BulletsCollected.Set(float64(len(bullets)))
	metrics.BulletsLastCollectedAt.Set(float64(time.Now().Unix()))

	log.Printf("Bullets collector: collected %d bullets → %s", len(bullets), c.filePath)
}
