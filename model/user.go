package model

import "time"

// User — структура для парсинга тела запроса на регистрацию
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// UpdatedUser — структура для запроса на обновление
// status_id не доступен через этот запрос
type UpdatedUser struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

// UpdateStatus — структура для запроса на обновление статуса
type UpdateStatus struct {
	ID       int `json:"id"`
	StatusID int `json:"status_id"`
}

// UserResponse — структура ответа с полными данными пользователя
type UserResponse struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Phone     string    `json:"phone"`
	StatusID  int       `json:"status_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
