package repository

import (
	"database/sql"
	"errors"
	"user-service/model"
)

// UserRepository — структура которая хранит соединение с БД
type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create — добавляет нового пользователя в БД со статусом 1 (на модерации)
func (r *UserRepository) Create(user model.User) (int, error) {
	var id int
	err := r.db.QueryRow(
		`INSERT INTO users (name, email, phone, status_id) 
		 VALUES ($1, $2, $3, 1) RETURNING id`,
		user.Name, user.Email, user.Phone,
	).Scan(&id)
	return id, err
}

// GetByID — возвращает пользователя по id
func (r *UserRepository) GetByID(id string) (model.UserResponse, error) {
	var user model.UserResponse
	err := r.db.QueryRow(
		`SELECT id, name, email, phone, status_id, created_at, updated_at 
		 FROM users WHERE id = $1`,
		id,
	).Scan(&user.ID, &user.Name, &user.Email, &user.Phone,
		&user.StatusID, &user.CreatedAt, &user.UpdatedAt)
	return user, err
}

// Update — обновляет email и/или телефон пользователя по id
// статус не меняется
func (r *UserRepository) Update(u model.UpdatedUser) (int64, error) {
	res, err := r.db.Exec(
		`UPDATE users 
		 SET email      = COALESCE(NULLIF($1, ''), email),
		     phone      = COALESCE(NULLIF($2, ''), phone),
		     updated_at = NOW()
		 WHERE id = $3`,
		u.Email, u.Phone, u.ID,
	)
	if err != nil {
		return 0, err
	}
	rows, err := res.RowsAffected()
	return rows, err
}

// UpdateStatus — меняет статус пользователя
// статус 3 можно поменять на любой
// статусы 1 и 2 можно менять между собой
// нельзя поставить статус 3 через эту ручку
func (r *UserRepository) UpdateStatus(u model.UpdateStatus) (int64, error) {
	// статус 3 нельзя установить через updateStatus
	if u.StatusID == 3 {
		return 0, errors.New("cannot set status 3 manually, use delete endpoint")
	}

	// статус должен быть 1 или 2
	if u.StatusID != 1 && u.StatusID != 2 {
		return 0, errors.New("invalid status_id, must be 1 or 2")
	}

	res, err := r.db.Exec(
		`UPDATE users 
		 SET status_id  = $1,
		     updated_at = NOW()
		 WHERE id = $2`,
		u.StatusID, u.ID,
	)
	if err != nil {
		return 0, err
	}
	rows, err := res.RowsAffected()
	return rows, err
}

// Delete — мягкое удаление: проставляем статус 3 вместо удаления записи
func (r *UserRepository) Delete(id string) (int64, error) {
	res, err := r.db.Exec(
		`UPDATE users 
		 SET status_id  = 3,
		     updated_at = NOW()
		 WHERE id = $1`,
		id,
	)
	if err != nil {
		return 0, err
	}
	rows, err := res.RowsAffected()
	return rows, err
}
