package configs

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	MySQL struct {
		Host     string
		Port     int
		User     string
		Password string
		Name     string
	}
	Redis struct {
		Host string
		Port int
		User string
	}
	Bot struct {
		Token string
	}
}

func LoadConfig() (Config, error) {
	var config Config

	if err := godotenv.Load(); err != nil {
		log.Print("No .env file found")
	}

	config.MySQL.Host = os.Getenv("MYSQL_HOST")
	config.MySQL.Port = getEnvAsInt("MYSQL_PORT", 3306)
	config.MySQL.User = os.Getenv("MYSQL_USER")
	config.MySQL.Password = os.Getenv("MYSQL_PASSWORD")
	config.MySQL.Name = os.Getenv("MYSQL_NAME")

	config.Redis.Host = os.Getenv("REDIS_HOST")
	config.Redis.Port = getEnvAsInt("REDIS_PORT", 6379)
	config.Redis.User = os.Getenv("REDIS_USER")

	config.Bot.Token = os.Getenv("BOT_TOKEN")

	return config, nil
}

// getEnvAsInt преобразует переменную окружения в int.
// Возвращает значение по умолчанию, если переменная не установлена или не может быть преобразована в int.
func getEnvAsInt(key string, defaultVal int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultVal
}
