package handlers

import (
	"encoding/json"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"io"
	"log"
	"net/http"
	"rutubeTest/pkg/sessions"
	"rutubeTest/pkg/user"
	"strings"
)

const (
	ErrReading      = `{"message": "error reading request"}`
	ErrUserNotFound = `{"message":"user not found"}`
	ErrInvalidPass  = `{"message":"invalid password"}`
	ErrBadRequest   = `{"message": "bad request"}`
)

type UserHandler struct {
	UserRepo user.UserRepo
	Logger   *zap.SugaredLogger
	Sessions sessions.SessionManagerInterface
}

type AuthForm struct {
	Username string `json:"username"`
	Password string `json:"password"  validate:"required"`
}

type RegForm struct {
	Username   string `json:"username"  validate:"required"`
	FirstName  string `json:"firstname"  validate:"required"`
	MiddleName string `json:"middlename"`
	LastName   string `json:"lastname"  validate:"required"`
	Password   string `json:"password"  validate:"required"`
	Birthday   string `json:"birthday"  validate:"required"`
	Telegram   string `json:"telegram"  validate:"required"`
}

type SubscribeForm struct {
	UserID       int64 `json:"userID"  validate:"required"`
	SubscriberID int64 `json:"subscriberID"  validate:"required"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	h.Logger.Infoln("Start logging")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, ErrReading, http.StatusBadRequest)
		return
	}
	r.Body.Close()

	af := &AuthForm{}
	if err = json.Unmarshal(body, af); err != nil {
		http.Error(w, ErrBadRequest, http.StatusBadRequest)
		return
	}

	h.Logger.Infoln("User data unmarshalled")

	// Валидация предоставленных данных
	errors := dataValidation(af)
	if errors != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		err = json.NewEncoder(w).Encode(map[string][]map[string]string{"errors": errors})
		if err != nil {
			h.Logger.Errorln(err.Error())
		}
		return
	}

	h.Logger.Infoln("User data validated")

	// Авторизация пользователя по предоставленным данным
	u, err := h.UserRepo.Authorize(af.Username, af.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err == user.ErrNoUser {
		http.Error(w, ErrUserNotFound, http.StatusUnauthorized)
		return
	}
	if err == user.ErrBadPass {
		http.Error(w, ErrInvalidPass, http.StatusUnauthorized)
		return
	}
	if u == nil {
		http.Error(w, ErrBadRequest, http.StatusBadRequest)
		return
	}

	h.Logger.Infoln("User authorized")

	// Сохранение сессии в redis.
	sess, err := h.Sessions.Create(&sessions.Session{
		ID:        u.ID,
		Login:     u.Username,
		Useragent: r.UserAgent(),
	})
	if err != nil {
		log.Println("cant create session:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(map[string]string{
		"session": sess.ID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(resp)
	if err != nil {
		h.Logger.Errorln(err.Error())
		return
	}
	h.Logger.Infoln("Response sent")
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	h.Logger.Infoln("Start registering")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, ErrReading, http.StatusBadRequest)
		return
	}
	r.Body.Close()

	rf := &RegForm{}
	if err = json.Unmarshal(body, rf); err != nil {
		http.Error(w, ErrBadRequest, http.StatusBadRequest)
		return
	}

	h.Logger.Infoln("User data unmarshalled")

	// Валидация предоставленных данных.
	errors := dataValidation(rf)
	if len(errors) > 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		err = json.NewEncoder(w).Encode(map[string][]map[string]string{"errors": errors})
		if err != nil {
			h.Logger.Errorln(err.Error())
		}
		return
	}

	h.Logger.Infoln("User data validated")

	// Создание пользователя по предоставленным данным.
	u, err := h.UserRepo.MakeUser(rf.Username, rf.Password, rf.FirstName, rf.MiddleName, rf.LastName, rf.Birthday, rf.Telegram)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// обработка ошибки, что юзер уже есть.
	if err == user.ErrExists {
		newError := map[string]string{
			"location": "body",
			"param":    rf.Username,
			"msg":      "already exists",
		}
		errors = append(errors, newError)
	}

	if errors != nil {
		w.WriteHeader(http.StatusUnprocessableEntity)
		err = json.NewEncoder(w).Encode(map[string][]map[string]string{"errors": errors})
		if err != nil {
			h.Logger.Errorln(err.Error())
		}
		return
	}

	h.Logger.Infoln("User made")

	// Сохранение сессии в redis.
	sess, err := h.Sessions.Create(&sessions.Session{
		ID:        u.ID,
		Login:     u.Username,
		Useragent: r.UserAgent(),
	})
	if err != nil {
		log.Println("cant create session:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	resp, err := json.Marshal(map[string]string{
		"session": sess.ID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(resp)
	if err != nil {
		h.Logger.Errorln(err.Error())
		return
	}
	h.Logger.Infoln("Response sent")
}

func dataValidation(fd interface{}) []map[string]string {
	if err := validator.New().Struct(fd); err != nil {
		var newErrors []map[string]string
		for _, someErr := range err.(validator.ValidationErrors) {
			newError := map[string]string{
				"location": "body",
				"param":    strings.ToLower(someErr.StructField()),
				"msg":      "is required",
			}
			newErrors = append(newErrors, newError)
		}
		return newErrors
	}

	return nil
}

func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	h.Logger.Infoln("Start authorization")

	token := r.Header.Get("Authorization")
	if !strings.HasPrefix(token, "Bearer ") {
		http.Error(w, ErrUserNotFound, http.StatusUnauthorized)
		return
	}

	sess := h.Sessions.Check(&sessions.SessionID{ID: token[7:]})
	if sess == nil {
		http.Error(w, ErrUserNotFound, http.StatusUnauthorized)
		return
	}

	users, err := h.UserRepo.GetUsers()
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.Logger.Infoln("users received")

	resp, err := json.Marshal(users)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = w.Write(resp)
	if err != nil {
		h.Logger.Errorln(err.Error())
		return
	}
	h.Logger.Infoln("Response sent")
}

func (h *UserHandler) SubscribeToUser(w http.ResponseWriter, r *http.Request) {
	h.Logger.Infoln("Start authorization")

	token := r.Header.Get("Authorization")
	if !strings.HasPrefix(token, "Bearer ") {
		http.Error(w, ErrUserNotFound, http.StatusUnauthorized)
		return
	}

	sess := h.Sessions.Check(&sessions.SessionID{ID: token[7:]})
	if sess == nil {
		http.Error(w, ErrUserNotFound, http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, ErrReading, http.StatusBadRequest)
		return
	}
	r.Body.Close()

	sf := &SubscribeForm{}
	if err = json.Unmarshal(body, sf); err != nil {
		http.Error(w, ErrBadRequest, http.StatusBadRequest)
		return
	}

	h.Logger.Infoln("User data unmarshalled")

	// Валидация предоставленных данных.
	errors := dataValidation(sf)
	if len(errors) > 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		err = json.NewEncoder(w).Encode(map[string][]map[string]string{"errors": errors})
		if err != nil {
			h.Logger.Errorln(err.Error())
		}
		return
	}

	h.Logger.Infoln("User data validated")

	_, err = h.UserRepo.Subscribe(sf.UserID, sf.SubscriberID, 1)
	if err != nil {
		http.Error(w, ErrBadRequest, http.StatusBadRequest)
		return
	}
}

func (h *UserHandler) UnsubscribeToUser(w http.ResponseWriter, r *http.Request) {
	h.Logger.Infoln("Start authorization")

	token := r.Header.Get("Authorization")
	if !strings.HasPrefix(token, "Bearer ") {
		http.Error(w, ErrUserNotFound, http.StatusUnauthorized)
		return
	}

	sess := h.Sessions.Check(&sessions.SessionID{ID: token[7:]})
	if sess == nil {
		http.Error(w, ErrUserNotFound, http.StatusUnauthorized)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, ErrReading, http.StatusBadRequest)
		return
	}
	r.Body.Close()

	sf := &SubscribeForm{}
	if err = json.Unmarshal(body, sf); err != nil {
		http.Error(w, ErrBadRequest, http.StatusBadRequest)
		return
	}

	h.Logger.Infoln("User data unmarshalled")

	// Валидация предоставленных данных.
	errors := dataValidation(sf)
	if len(errors) > 0 {
		w.WriteHeader(http.StatusUnprocessableEntity)
		err = json.NewEncoder(w).Encode(map[string][]map[string]string{"errors": errors})
		if err != nil {
			h.Logger.Errorln(err.Error())
		}
		return
	}

	h.Logger.Infoln("User data validated")

	_, err = h.UserRepo.Subscribe(sf.UserID, sf.SubscriberID, 0)
	if err != nil {
		http.Error(w, ErrBadRequest, http.StatusBadRequest)
		return
	}
}
