package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"
	"user-service/cache"
	"user-service/metrics"
	"user-service/model"
	"user-service/repository"
)

// UserHandler — структура которая хранит репозиторий и кэш
type UserHandler struct {
	repo  *repository.UserRepository
	cache *cache.Cache
}

func NewUserHandler(repo *repository.UserRepository, cache *cache.Cache) *UserHandler {
	return &UserHandler{repo: repo, cache: cache}
}

// @Summary      Регистрация пользователя
// @Description  Создаёт нового пользователя в БД
// @Accept       json
// @Produce      json
// @Param        user  body      model.User  true  "Данные пользователя"
// @Success      200   {object}  map[string]int
// @Failure      400   {string}  string  "Bad request"
// @Failure      500   {string}  string  "DB error"
// @Router       /user/register [post]
func (h *UserHandler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var user model.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		metrics.RequestsTotal.WithLabelValues("register", "400").Inc()
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	id, err := h.repo.Create(user)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			metrics.RequestsTotal.WithLabelValues("register", "409").Inc()
			http.Error(w, "Email already exists", http.StatusConflict)
			return
		}
		metrics.RequestsTotal.WithLabelValues("register", "500").Inc()
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	metrics.RequestsTotal.WithLabelValues("register", "200").Inc()
	metrics.RequestDuration.WithLabelValues("register").Observe(time.Since(start).Seconds())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]int{"id": id})
}

// @Summary      Обновление пользователя
// @Description  Обновляет email и/или телефон пользователя по id
// @Accept       json
// @Produce      json
// @Param        user  body      model.UpdatedUser  true  "Данные для обновления"
// @Success      200   {object}  map[string]string
// @Failure      400   {string}  string  "Bad request"
// @Failure      404   {string}  string  "User not found"
// @Failure      500   {string}  string  "DB error"
// @Router       /user/update [patch]
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var u model.UpdatedUser
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		metrics.RequestsTotal.WithLabelValues("update", "400").Inc()
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	rows, err := h.repo.Update(u)
	if err != nil {
		metrics.RequestsTotal.WithLabelValues("update", "500").Inc()
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if rows == 0 {
		metrics.RequestsTotal.WithLabelValues("update", "404").Inc()
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// инвалидируем кэш — данные пользователя изменились
	h.cache.Delete("user:" + strconv.Itoa(u.ID))

	metrics.RequestsTotal.WithLabelValues("update", "200").Inc()
	metrics.RequestDuration.WithLabelValues("update").Observe(time.Since(start).Seconds())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}

// @Summary      Получение пользователя
// @Description  Возвращает данные пользователя по id
// @Produce      json
// @Param        id   query     int  true  "ID пользователя"
// @Success      200  {object}  model.UserResponse
// @Failure      400  {string}  string  "Bad request"
// @Failure      404  {string}  string  "User not found"
// @Failure      500  {string}  string  "DB error"
// @Router       /user/get [get]
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		metrics.RequestsTotal.WithLabelValues("get", "400").Inc()
		http.Error(w, "Bad request: id is required", http.StatusBadRequest)
		return
	}

	// сначала смотрим в кэш — не идём в БД если данные свежие
	if cached, ok := h.cache.Get("user:" + idStr); ok {
		metrics.RequestsTotal.WithLabelValues("get", "200").Inc()
		metrics.RequestDuration.WithLabelValues("get").Observe(time.Since(start).Seconds())
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cached))
		return
	}

	// кэша нет — идём в БД
	user, err := h.repo.GetByID(idStr)
	if err == sql.ErrNoRows {
		metrics.RequestsTotal.WithLabelValues("get", "404").Inc()
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	if err != nil {
		metrics.RequestsTotal.WithLabelValues("get", "500").Inc()
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// сохраняем в кэш
	data, _ := json.Marshal(user)
	h.cache.Set("user:"+idStr, string(data))

	metrics.RequestsTotal.WithLabelValues("get", "200").Inc()
	metrics.RequestDuration.WithLabelValues("get").Observe(time.Since(start).Seconds())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// @Summary      Удаление пользователя
// @Description  Удаляет пользователя по id (soft delete — статус 3)
// @Produce      json
// @Param        id   query     int  true  "ID пользователя"
// @Success      200  {object}  map[string]string
// @Failure      400  {string}  string  "Bad request"
// @Failure      404  {string}  string  "User not found"
// @Failure      500  {string}  string  "DB error"
// @Router       /user/delete [delete]
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		metrics.RequestsTotal.WithLabelValues("delete", "400").Inc()
		http.Error(w, "Bad request: id is required", http.StatusBadRequest)
		return
	}

	rows, err := h.repo.Delete(idStr)
	if err != nil {
		metrics.RequestsTotal.WithLabelValues("delete", "500").Inc()
		http.Error(w, "DB error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if rows == 0 {
		metrics.RequestsTotal.WithLabelValues("delete", "404").Inc()
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// инвалидируем кэш — статус пользователя изменился
	h.cache.Delete("user:" + idStr)

	metrics.RequestsTotal.WithLabelValues("delete", "200").Inc()
	metrics.RequestDuration.WithLabelValues("delete").Observe(time.Since(start).Seconds())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}

// @Summary      Обновление статуса пользователя
// @Description  Меняет статус пользователя: 1 (модерация) ↔ 2 (активный). Статус 3 только через delete
// @Accept       json
// @Produce      json
// @Param        user  body      model.UpdateStatus  true  "Данные для обновления статуса"
// @Success      200   {object}  map[string]string
// @Failure      400   {string}  string  "Bad request"
// @Failure      404   {string}  string  "User not found"
// @Failure      500   {string}  string  "DB error"
// @Router       /user/updateStatus [patch]
func (h *UserHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	if r.Method != http.MethodPatch {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var u model.UpdateStatus
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		metrics.RequestsTotal.WithLabelValues("updateStatus", "400").Inc()
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	rows, err := h.repo.UpdateStatus(u)
	if err != nil {
		metrics.RequestsTotal.WithLabelValues("updateStatus", "400").Inc()
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if rows == 0 {
		metrics.RequestsTotal.WithLabelValues("updateStatus", "404").Inc()
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}

	// инвалидируем кэш — статус пользователя изменился
	h.cache.Delete("user:" + strconv.Itoa(u.ID))

	metrics.RequestsTotal.WithLabelValues("updateStatus", "200").Inc()
	metrics.RequestDuration.WithLabelValues("updateStatus").Observe(time.Since(start).Seconds())

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
}
