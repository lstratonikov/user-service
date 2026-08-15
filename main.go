package main

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"user-service/cache"
	_ "user-service/docs"
	"user-service/handler"
	"user-service/loadtest"
	"user-service/metrics"
	"user-service/repository"
	"user-service/simulator"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger"
)

// @title           User Service API
// @version         1.0
// @description     Сервис регистрации и обновления пользователей
// @host            localhost:8080
// @BasePath        /
func main() {
	// настраиваем логирование в файл и stdout одновременно
	logFile, err := os.OpenFile("/tmp/user-service.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal("Failed to open log file:", err)
	}
	defer logFile.Close()

	// MultiWriter пишет одновременно в терминал и в файл
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)

	// читаем хост БД из переменной окружения
	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}

	connStr := fmt.Sprintf(
		"host=%s user=lstratonikov password=Udfhltqcrfz25%% dbname=testdb sslmode=disable default_query_exec_mode=simple_protocol",
		dbHost,
	)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatal("Failed to connect to DB:", err)
	}
	defer db.Close()

	// пул соединений
	db.SetMaxOpenConns(200)
	db.SetMaxIdleConns(200)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id         SERIAL PRIMARY KEY,
		name       TEXT NOT NULL,
		email      TEXT NOT NULL UNIQUE,
		phone      TEXT,
		status_id  INT DEFAULT 1,
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	)`)
	if err != nil {
		log.Fatal("Failed to create table:", err)
	}

	// запускаем сборщик метрик БД каждые 15 секунд
	metrics.StartDBMetricsCollector(db)

	repo := repository.NewUserRepository(db)

	// кэш с TTL 30 секунд
	userCache := cache.NewCache(30 * time.Second)
	h := handler.NewUserHandler(repo, userCache)

	// запускаем сборщик патронов
	collector := loadtest.NewBulletsCollector(db, "/tmp/bullets.json")
	collector.Start()

	http.HandleFunc("/user/register", h.RegisterUser)
	http.HandleFunc("/user/update", h.UpdateUser)
	http.HandleFunc("/user/get", h.GetUser)
	http.HandleFunc("/user/delete", h.DeleteUser)
	http.HandleFunc("/user/updateStatus", h.UpdateStatus)

	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/swagger/", httpSwagger.WrapHandler)

	http.HandleFunc("/admin/test-start", func(w http.ResponseWriter, r *http.Request) {
		metrics.LoadTestRunning.Set(1)
		log.Println("Load test started")
		w.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/admin/test-stop", func(w http.ResponseWriter, r *http.Request) {
		metrics.LoadTestRunning.Set(0)
		metrics.BulletsCollected.Set(0)
		log.Println("Load test stopped")
		w.WriteHeader(http.StatusOK)
	})

	// ручной запуск сборки патронов
	http.HandleFunc("/admin/collect-bullets", func(w http.ResponseWriter, r *http.Request) {
		go collector.Collect()
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Bullets collection started")
	})

	// запускаем симулятор — 5 RPS фоновой нагрузки
	sim := simulator.NewSimulator("http://localhost:8080", 5)
	sim.Start()

	log.Println("Server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
