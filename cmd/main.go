package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"log"
	"net/http"
	"rutubeTest/bot"
	"rutubeTest/configs"
	"rutubeTest/pkg/handlers"
	"rutubeTest/pkg/middleware"
	"rutubeTest/pkg/sessions"
	"rutubeTest/pkg/user"
)

func main() {
	config, err := configs.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	// Настраиваем подключение к mysql.
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		config.MySQL.User,
		config.MySQL.Password,
		config.MySQL.Host,
		config.MySQL.Port,
		config.MySQL.Name)

	mysql, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Printf("Error opening connection to database: %v", err)
	}

	err = mysql.Ping()
	if err != nil {
		log.Printf("Error connecting to database: %v", err)
	}
	log.Println("Успешное подключение к MySQL!")

	// Настраиваем подключение к redis
	redisAddr := fmt.Sprintf("redis://%s:@%s:%d/0", config.Redis.User, config.Redis.Host, config.Redis.Port)
	addr := flag.String("addr", redisAddr, "help message for flagname")
	redisConn, err := redis.DialURL(*addr)
	if err != nil {
		log.Fatalf("Error connecting to redis: %v", err)
	}
	log.Println("Успешное подключение к Redis!")

	sessManager := sessions.NewSessionManager(redisConn)

	zapLogger, err := zap.NewProduction()
	if err != nil {
		log.Printf("Error making new logger: %v", err)
	}
	defer func() {
		if err = zapLogger.Sync(); err != nil {
			log.Printf("Failed to sync logger: %v", err)
		}
	}()
	logger := zapLogger.Sugar()

	userRepo := user.NewMysqlRepo(mysql)

	userHandler := &handlers.UserHandler{
		UserRepo: userRepo,
		Logger:   logger,
		Sessions: sessManager,
	}

	r := mux.NewRouter()

	r.HandleFunc("/api/login", userHandler.Login).Methods("POST")
	r.HandleFunc("/api/register", userHandler.Register).Methods("POST")
	r.HandleFunc("/api/users", userHandler.GetUsers).Methods("GET")
	r.HandleFunc("/api/subscribe", userHandler.SubscribeToUser).Methods("POST")
	r.HandleFunc("/api/unsubscribe", userHandler.UnsubscribeToUser).Methods("POST")

	middleWares := middleware.AccessLog(logger, r)

	// Запуск веб-сервиса в горутине
	go func() {
		log.Println("starting server at :8080")
		err = http.ListenAndServe(":8080", middleWares)
		if err != nil {
			panic(err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Запуск тг бота в горутине
	go func() {
		err = bot.StartTaskBot(ctx, config.Bot.Token, userRepo)
		if err != nil {
			log.Println(err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")
}
