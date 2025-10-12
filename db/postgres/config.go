package postgres

import "time"

// Низкоуровневый конфиг подключения (host/port/...).
// Удобен в приложениях: читаем ENV → передаём сюда → остальное сделает библиотека.
type DBConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int           // максимум коннекций в пуле
	MaxIdleConns    int           // минимум (минимальный размер пула)
	ConnMaxLifetime time.Duration // переоткрывать коннект не реже чем
	ConnMaxIdleTime time.Duration // держать idle не дольше чем
}

// Высокоуровневый конфиг — через готовый URL.
// Оставляем для беквард-совместимости и особых случаев (например, сложные DSN).
type Config struct {
	URL    string            // postgres://user:pass@host:port/dbname?sslmode=disable
	Params map[string]string // доп.параметры для URL (перезаписывают query)
	// Параметры пула
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}
