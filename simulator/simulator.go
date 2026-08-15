package simulator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// Simulator — генератор фоновой нагрузки
// Имитирует реальных пользователей пока сервис работает
type Simulator struct {
	baseURL string
	rps     int // запросов в секунду
}

func NewSimulator(baseURL string, rps int) *Simulator {
	return &Simulator{baseURL: baseURL, rps: rps}
}

// Start — запускает симулятор в фоновом горутине
// Не блокирует основной поток
func (s *Simulator) Start() {
	go s.run()
	log.Printf("Simulator started: %d RPS background traffic", s.rps)
}

func (s *Simulator) run() {
	// ticker срабатывает каждые (1000/rps) миллисекунд
	ticker := time.NewTicker(time.Duration(1000/s.rps) * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		// каждый тик — один случайный флоу пользователя в горутине
		go s.userFlow()
	}
}

// userFlow — один цикл жизни пользователя:
// регистрация → получение → обновление → удаление
func (s *Simulator) userFlow() {
	// регистрируем пользователя
	id, err := s.register()
	if err != nil {
		return
	}

	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

	// получаем данные
	s.get(id)
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

	// обновляем телефон
	s.update(id)
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

	//обновляем статус
	s.updateStatus(id, 2)
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)

	s.delete(id)
}

func (s *Simulator) register() (int, error) {
	body, _ := json.Marshal(map[string]string{
		"name":  fmt.Sprintf("User %d", rand.Intn(100000)),
		"email": fmt.Sprintf("user_%d@test.com", rand.Intn(100000000)),
		"phone": fmt.Sprintf("+7999%07d", rand.Intn(9000000)),
	})

	resp, err := http.Post(
		s.baseURL+"/user/register",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result map[string]int
	json.NewDecoder(resp.Body).Decode(&result)
	return result["id"], nil
}

func (s *Simulator) get(id int) {
	resp, err := http.Get(fmt.Sprintf("%s/user/get?id=%d", s.baseURL, id))
	if err != nil {
		return
	}
	resp.Body.Close()
}

func (s *Simulator) update(id int) {
	body, _ := json.Marshal(map[string]interface{}{
		"id":    id,
		"phone": fmt.Sprintf("+7999%07d", rand.Intn(9000000)),
	})

	req, _ := http.NewRequest(http.MethodPatch, s.baseURL+"/user/update", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func (s *Simulator) updateStatus(id int, status int) {
	body, _ := json.Marshal(map[string]interface{}{
		"id":        id,
		"status_id": status,
	})

	req, _ := http.NewRequest(http.MethodPatch, s.baseURL+"/user/updateStatus", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}

func (s *Simulator) delete(id int) {
	req, _ := http.NewRequest(
		http.MethodDelete,
		fmt.Sprintf("%s/user/delete?id=%d", s.baseURL, id),
		nil,
	)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	resp.Body.Close()
}
