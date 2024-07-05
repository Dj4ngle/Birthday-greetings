package bot

import (
	"context"
	"fmt"
	tgbotapi "github.com/skinass/telegram-bot-api/v5"
	"log"
	"net/http"
	"os"
	"rutubeTest/pkg/user"
	"strconv"
	"strings"
	"time"
)

var (
	// Нужен для работы тг бота в локалке
	WebhookURL = "https://5f1f-188-32-207-71.ngrok-free.app"

	commandHandlers = map[string]func(tgbotapi.Update, *user.UserMysqlRepository) []tgbotapi.MessageConfig{
		"/subscribe":   subscribeHandler,
		"/unsubscribe": unsubscribeHandler,
		"/start":       startHandler,
		"/users":       usersListHandler,
	}
)

func usersListHandler(update tgbotapi.Update, userRepo *user.UserMysqlRepository) []tgbotapi.MessageConfig {
	users, err := userRepo.GetUsers()
	if err != nil {
		return nil
	}

	messages := make([]tgbotapi.MessageConfig, 0, len(users)) // Предварительное выделение памяти с нужным размером

	var msg tgbotapi.MessageConfig
	var str string
	for _, u := range users {
		str = "ID: " + strconv.FormatInt(u.ID, 10) +
			" ФИО: " + u.FirstName + " " + u.MiddleName + " " + u.LastName +
			" " + u.Birthday + " " + u.Telegram
		msg = tgbotapi.NewMessage(
			update.Message.Chat.ID,
			str,
		)
		messages = append(messages, msg)
	}

	return messages
}

func startHandler(update tgbotapi.Update, userRepo *user.UserMysqlRepository) []tgbotapi.MessageConfig {
	err := userRepo.UpdateUser(update.Message.From.ID, update.Message.From.UserName)
	if err != nil {
		msg := tgbotapi.NewMessage(
			update.Message.Chat.ID,
			err.Error(),
		)
		return []tgbotapi.MessageConfig{msg}
	}
	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		"Добро пожаловать. Напишите /users, чтобы увидеть всех пользователей.\n"+
			"Напишите /subscribe или /unsubscribe, а после id для подписки отписки на пользователя.\n"+
			"Например, /subscribe 1",
	)
	return []tgbotapi.MessageConfig{msg}
}

func subscribeHandler(update tgbotapi.Update, userRepo *user.UserMysqlRepository) []tgbotapi.MessageConfig {
	userID, err := strconv.Atoi(update.Message.Text[11:])
	if err != nil {
		msg := tgbotapi.NewMessage(
			update.Message.Chat.ID,
			err.Error(),
		)
		return []tgbotapi.MessageConfig{msg}
	}

	u, err := userRepo.GetUserByTelegram("@" + update.Message.From.UserName)
	if err != nil {
		msg := tgbotapi.NewMessage(
			update.Message.Chat.ID,
			err.Error(),
		)
		return []tgbotapi.MessageConfig{msg}
	}

	subUser, err := userRepo.Subscribe(int64(userID), u.ID, 1)
	if err != nil {
		msg := tgbotapi.NewMessage(
			update.Message.Chat.ID,
			err.Error(),
		)
		return []tgbotapi.MessageConfig{msg}
	}

	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		"Вы подписались на "+subUser.Telegram,
	)
	return []tgbotapi.MessageConfig{msg}
}

func unsubscribeHandler(update tgbotapi.Update, userRepo *user.UserMysqlRepository) []tgbotapi.MessageConfig {
	userID, err := strconv.Atoi(update.Message.Text[13:])
	if err != nil {
		msg := tgbotapi.NewMessage(
			update.Message.Chat.ID,
			err.Error(),
		)
		return []tgbotapi.MessageConfig{msg}
	}

	u, err := userRepo.GetUserByTelegram("@" + update.Message.From.UserName)
	if err != nil {
		msg := tgbotapi.NewMessage(
			update.Message.Chat.ID,
			err.Error(),
		)
		return []tgbotapi.MessageConfig{msg}
	}

	subUser, err := userRepo.Subscribe(int64(userID), u.ID, 0)
	if err != nil {
		msg := tgbotapi.NewMessage(
			update.Message.Chat.ID,
			err.Error(),
		)
		return []tgbotapi.MessageConfig{msg}
	}

	msg := tgbotapi.NewMessage(
		update.Message.Chat.ID,
		"Вы отписались от "+subUser.Telegram,
	)
	return []tgbotapi.MessageConfig{msg}
}

func updateHandler(update tgbotapi.Update, userRepo *user.UserMysqlRepository) []tgbotapi.MessageConfig {
	if update.Message == nil {
		return nil // Нет сообщения для обработки
	}

	text := update.Message.Text
	for cmd, handler := range commandHandlers {
		if strings.HasPrefix(text, cmd) {
			return handler(update, userRepo)
		}
	}

	return nil
}

func StartTaskBot(ctx context.Context, botToken string, userRepo *user.UserMysqlRepository) error {

	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Printf("NewBotAPI failed: %s", err)
		return err
	}

	bot.Debug = true
	fmt.Printf("Authorized on account %s\n", bot.Self.UserName)

	wh, err := tgbotapi.NewWebhook(WebhookURL)
	if err != nil {
		log.Printf("NewWebhook failed: %s", err)
		return err
	}

	_, err = bot.Request(wh)
	if err != nil {
		log.Printf("SetWebhook failed: %s", err)
		return err
	}

	updates := bot.ListenForWebhook("/")

	http.HandleFunc("/state", func(w http.ResponseWriter, r *http.Request) {
		_, err = w.Write([]byte("all is working"))
		if err != nil {
			return
		}
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	go func() {
		log.Fatalln("http err:", http.ListenAndServe(":"+port, nil))
	}()
	fmt.Println("start listen :" + port)

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Запуск сервиса уведомлений в горутине
	go func() {
		for {
			select {
			case <-ticker.C:
				CheckAndSendNotifications(userRepo, bot)
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case update := <-updates:
			log.Printf("upd: %#v\n", update)
			messages := updateHandler(update, userRepo)
			for _, v := range messages {
				_, err = bot.Send(v)
				if err != nil {
					return err
				}
			}
		case <-ctx.Done():

			if ctx.Err() == context.Canceled {
				log.Println("Operation was canceled")
				return nil
			}
			return ctx.Err()
		}
	}
}

func CheckAndSendNotifications(userRepo *user.UserMysqlRepository, bot *tgbotapi.BotAPI) {
	today := time.Now()
	month := int(today.Month())
	day := today.Day()

	users, err := userRepo.GetUserByBirthday(month, day)
	if err != nil {
		fmt.Println("Error fetching users:", err)
		return
	}

	for _, u := range users {
		subscribers, err := userRepo.GetSubscribedUsers(u.ID)
		if err != nil {
			fmt.Println("Error fetching subscribers:", err)
			continue
		}

		var str string
		for _, sub := range subscribers {
			str = u.FirstName + " " + u.MiddleName + " " + u.LastName
			sendTelegramNotification(bot, sub.TelegramID, str)
		}
	}
}

func sendTelegramNotification(bot *tgbotapi.BotAPI, chatID int64, employeeName string) {
	msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("Сегодня день рождения у %s! Поздравьте его!", employeeName))
	if _, err := bot.Send(msg); err != nil {
		fmt.Println("Error sending Telegram message:", err)
	}
}
