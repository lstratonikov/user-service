package metrics

import (
	"database/sql"
	"log"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// DBSize — размер базы данных в байтах
var DBSize = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "db_size_bytes",
		Help: "Размер базы данных в байтах",
	},
)

// DBTableSize — размер таблицы users в байтах
var DBTableSize = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "db_table_users_size_bytes",
		Help: "Размер таблицы users в байтах",
	},
)

// DBRowsCount — количество строк в таблице users
var DBRowsCount = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "db_table_users_rows_total",
		Help: "Количество строк в таблице users",
	},
)

// DBDiskTotal — общий размер диска в байтах
var DBDiskTotal = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "db_disk_total_bytes",
		Help: "Общий размер диска в байтах",
	},
)

// DBDiskFree — свободное место на диске в байтах
var DBDiskFree = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "db_disk_free_bytes",
		Help: "Свободное место на диске в байтах",
	},
)

func init() {
	prometheus.MustRegister(DBSize)
	prometheus.MustRegister(DBTableSize)
	prometheus.MustRegister(DBRowsCount)
	prometheus.MustRegister(DBDiskTotal)
	prometheus.MustRegister(DBDiskFree)
}

// StartDBMetricsCollector — запускает сборщик метрик БД в фоне
// собирает метрики каждые 15 секунд
func StartDBMetricsCollector(db *sql.DB) {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			collectDBMetrics(db)
			<-ticker.C
		}
	}()
	log.Println("DB metrics collector started")
}

// collectDiskMetrics — собирает метрики диска через системный вызов
func collectDiskMetrics() {
	var stat syscall.Statfs_t
	err := syscall.Statfs("/", &stat)
	if err != nil {
		log.Printf("Failed to get disk stats: %v", err)
		return
	}

	// общий размер диска
	total := float64(stat.Blocks) * float64(stat.Bsize)
	// свободное место
	free := float64(stat.Bfree) * float64(stat.Bsize)

	DBDiskTotal.Set(total)
	DBDiskFree.Set(free)
}

func collectDBMetrics(db *sql.DB) {
	// сначала собираем дисковые метрики
	collectDiskMetrics()

	// размер всей базы данных
	var dbSize float64
	err := db.QueryRow("SELECT pg_database_size(current_database())").Scan(&dbSize)
	if err != nil {
		log.Printf("Failed to get db size: %v", err)
		return
	}
	DBSize.Set(dbSize)

	// размер таблицы users включая индексы
	var tableSize float64
	err = db.QueryRow("SELECT pg_total_relation_size('users')").Scan(&tableSize)
	if err != nil {
		log.Printf("Failed to get table size: %v", err)
		return
	}
	DBTableSize.Set(tableSize)

	// количество строк в таблице users
	var rowsCount float64
	err = db.QueryRow("SELECT COUNT(*) FROM users").Scan(&rowsCount)
	if err != nil {
		log.Printf("Failed to get rows count: %v", err)
		return
	}
	DBRowsCount.Set(rowsCount)
}
