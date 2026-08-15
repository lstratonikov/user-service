package metrics

import "github.com/prometheus/client_golang/prometheus"

// RequestsTotal — счётчик запросов по каждому endpoint
var RequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Общее количество HTTP запросов",
	},
	[]string{"handler", "status"},
)

// RequestDuration — гистограмма времени ответа по каждому endpoint
var RequestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Время обработки HTTP запросов",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"handler"},
)

// BulletsCollected — количество патронов в последней сборке
var BulletsCollected = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "bullets_collected_total",
		Help: "Количество патронов собранных в последней сборке",
	},
)

// BulletsLastCollectedAt — unix timestamp последней сборки патронов
var BulletsLastCollectedAt = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "bullets_last_collected_at",
		Help: "Время последней сборки патронов (unix timestamp)",
	},
)

// LoadTestRunning — флаг запущен ли нагрузочный тест
var LoadTestRunning = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "load_test_running",
		Help: "1 если нагрузочный тест запущен, 0 если нет",
	},
)

func init() {
	prometheus.MustRegister(RequestsTotal)
	prometheus.MustRegister(RequestDuration)
	prometheus.MustRegister(BulletsCollected)
	prometheus.MustRegister(BulletsLastCollectedAt)
	prometheus.MustRegister(LoadTestRunning)
}
