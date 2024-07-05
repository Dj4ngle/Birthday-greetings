package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"io"
	"net/http"
	"net/http/httptest"
	"rutubeTest/pkg/sessions"
	"rutubeTest/pkg/user"
	"testing"
)

type ErrReader struct{}

func (e *ErrReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated read error")
}

type ErrorWriter struct {
	http.ResponseWriter
	Err error
}

func (ew *ErrorWriter) Write(p []byte) (int, error) {
	return 0, ew.Err
}

func TestLoginHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := user.NewMockUserRepo(ctrl)
	mockSessions := sessions.NewMockSessionManagerInterface(ctrl)
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("Got err when making")
		return
	}

	service := &UserHandler{
		UserRepo: mockRepo,
		Logger:   logger.Sugar(),
		Sessions: mockSessions,
	}

	tests := []struct {
		name          string
		setupMocks    func()
		requestBody   interface{}
		requestReader io.Reader
		wantStatus    int
		expectError   bool
		customWriter  bool
	}{
		{
			name: "Успешный login",
			setupMocks: func() {
				mockRepo.EXPECT().Authorize("validUser", "validPass").Return(&user.User{}, nil)
				mockSessions.EXPECT().Create(gomock.Any()).Return(&sessions.SessionID{ID: "session-id"}, nil)
			},
			requestBody: map[string]string{"username": "validUser", "password": "validPass"},
			wantStatus:  http.StatusOK,
			expectError: false,
		},
		{
			name:          "Проверка обработки ошибки при io.ReadAll(r.Body)",
			setupMocks:    func() {},
			requestBody:   map[string]int{"username": 123, "password": 456},
			wantStatus:    http.StatusBadRequest,
			expectError:   true,
			requestReader: &ErrReader{},
		},
		{
			name:        "Проверка обработки ошибки при валидации",
			setupMocks:  func() {},
			requestBody: map[string]string{"username": "", "password": ""},
			wantStatus:  http.StatusUnprocessableEntity,
			expectError: true,
		},
		{
			name: "Проверка обработки ошибки при авторизации, что юзер не найден",
			setupMocks: func() {
				mockRepo.EXPECT().Authorize("invalidUser", "invalidPass").Return(nil, user.ErrNoUser)
			},
			requestBody: map[string]string{"username": "invalidUser", "password": "invalidPass"},
			wantStatus:  http.StatusBadRequest,
			expectError: true,
		},
		{
			name: "Проверка обработки ошибки при авторизации, что пароль неправильный",
			setupMocks: func() {
				mockRepo.EXPECT().Authorize("someUser", "badPass").Return(nil, user.ErrBadPass)
			},
			requestBody: map[string]string{"username": "someUser", "password": "badPass"},
			wantStatus:  http.StatusBadRequest,
			expectError: true,
		},
		{
			name: "Обработка ошибки при создании сессии",
			setupMocks: func() {
				mockRepo.EXPECT().Authorize("validUser", "validPass").Return(&user.User{}, nil)
				mockSessions.EXPECT().Create(gomock.Any()).Return(nil, fmt.Errorf("session creation failed"))
			},
			requestBody: map[string]string{"username": "validUser", "password": "validPass"},
			wantStatus:  http.StatusInternalServerError,
			expectError: true,
		},
		{
			name: "Обработка ошибки при создании ответа",
			setupMocks: func() {
				mockRepo.EXPECT().Authorize("validUser", "validPass").Return(&user.User{}, nil)
				mockSessions.EXPECT().Create(gomock.Any()).Return(&sessions.SessionID{ID: "session-id"}, nil)
			},
			requestBody:  map[string]string{"username": "validUser", "password": "validPass"},
			expectError:  true,
			customWriter: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks()

			var req *http.Request
			if tc.requestReader != nil {
				req = httptest.NewRequest("POST", "/api/login", tc.requestReader)
			} else {
				body, err := json.Marshal(tc.requestBody)
				assert.NoError(t, err)

				req = httptest.NewRequest("POST", "/api/login", bytes.NewReader(body))
			}

			if tc.customWriter {
				ew := &ErrorWriter{
					ResponseWriter: httptest.NewRecorder(),
					Err:            fmt.Errorf("simulated write error"),
				}

				service.Login(ew, req)
			} else {
				w := httptest.NewRecorder()

				service.Login(w, req)

				resp := w.Result()
				assert.Equal(t, tc.wantStatus, resp.StatusCode)

				if tc.expectError {
					assert.NotEqual(t, http.StatusOK, resp.StatusCode)
				} else {
					assert.Equal(t, http.StatusOK, resp.StatusCode)
				}
			}
		})
	}
}

func TestRegisterHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := user.NewMockUserRepo(ctrl)
	mockSessions := sessions.NewMockSessionManagerInterface(ctrl)
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("Got err when making")
		return
	}

	service := &UserHandler{
		UserRepo: mockRepo,
		Logger:   logger.Sugar(),
		Sessions: mockSessions,
	}

	tests := []struct {
		name          string
		setupMocks    func()
		requestBody   interface{}
		requestReader io.Reader
		wantStatus    int
		expectError   bool
		customWriter  bool
	}{
		{
			name: "Успешный register",
			setupMocks: func() {
				mockRepo.EXPECT().MakeUser("validUser", "validPass", "firstname",
					"middlename", "lastname", "2001-11-11", "@testuser").
					Return(&user.User{}, nil)
				mockSessions.EXPECT().Create(gomock.Any()).Return(&sessions.SessionID{ID: "session-id"}, nil)
			},
			requestBody: map[string]string{"username": "validUser", "password": "validPass", "firstname": "firstname",
				"middlename": "middlename", "lastname": "lastname", "birthday": "2001-11-11", "telegram": "@testuser"},
			wantStatus:  http.StatusOK,
			expectError: false,
		},
		{
			name:          "Проверка обработки ошибки при io.ReadAll(r.Body)",
			setupMocks:    func() {},
			requestBody:   map[string]int{"username": 123, "password": 456},
			wantStatus:    http.StatusBadRequest,
			expectError:   true,
			requestReader: &ErrReader{},
		},
		{
			name:        "Проверка обработки ошибки при валидации",
			setupMocks:  func() {},
			requestBody: map[string]string{"username": "", "password": ""},
			wantStatus:  http.StatusUnprocessableEntity,
			expectError: true,
		},
		{
			name: "Проверка обработки ошибки при авторизации, что юзер уже есть",
			setupMocks: func() {
				mockRepo.EXPECT().MakeUser("invalidUser", "invalidPass", "firstname",
					"middlename", "lastname", "2001-11-11", "@testuser").
					Return(&user.User{}, nil).Return(nil, user.ErrExists)
			},
			requestBody: map[string]string{"username": "invalidUser", "password": "invalidPass", "firstname": "firstname",
				"middlename": "middlename", "lastname": "lastname", "birthday": "2001-11-11", "telegram": "@testuser"},
			wantStatus:  http.StatusBadRequest,
			expectError: true,
		},
		{
			name: "Обработка ошибки при создании сессии",
			setupMocks: func() {
				mockRepo.EXPECT().MakeUser("validUser", "validPass", "firstname",
					"middlename", "lastname", "2001-11-11", "@testuser").
					Return(&user.User{}, nil).Return(&user.User{}, nil)
				mockSessions.EXPECT().Create(gomock.Any()).Return(nil, fmt.Errorf("session creation failed"))
			},
			requestBody: map[string]string{"username": "validUser", "password": "validPass", "firstname": "firstname",
				"middlename": "middlename", "lastname": "lastname", "birthday": "2001-11-11", "telegram": "@testuser"},
			wantStatus:  http.StatusInternalServerError,
			expectError: true,
		},
		{
			name: "Обработка ошибки при создании ответа",
			setupMocks: func() {
				mockRepo.EXPECT().MakeUser("validUser", "validPass", "firstname",
					"middlename", "lastname", "2001-11-11", "@testuser").
					Return(&user.User{}, nil).Return(&user.User{}, nil)
				mockSessions.EXPECT().Create(gomock.Any()).Return(&sessions.SessionID{ID: "session-id"}, nil)
			},
			requestBody: map[string]string{"username": "validUser", "password": "validPass", "firstname": "firstname",
				"middlename": "middlename", "lastname": "lastname", "birthday": "2001-11-11", "telegram": "@testuser"},
			expectError:  true,
			customWriter: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks()

			var req *http.Request
			if tc.requestReader != nil {
				req = httptest.NewRequest("POST", "/api/register", tc.requestReader)
			} else {
				body, err := json.Marshal(tc.requestBody)
				assert.NoError(t, err)

				req = httptest.NewRequest("POST", "/api/register", bytes.NewReader(body))
			}

			if tc.customWriter {
				ew := &ErrorWriter{
					ResponseWriter: httptest.NewRecorder(),
					Err:            fmt.Errorf("simulated write error"),
				}

				service.Register(ew, req)
			} else {
				w := httptest.NewRecorder()

				service.Register(w, req)

				resp := w.Result()
				assert.Equal(t, tc.wantStatus, resp.StatusCode)

				if tc.expectError {
					assert.NotEqual(t, http.StatusOK, resp.StatusCode)
				} else {
					assert.Equal(t, http.StatusOK, resp.StatusCode)
				}
			}
		})
	}
}

func TestGetUsersHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := user.NewMockUserRepo(ctrl)
	mockSessions := sessions.NewMockSessionManagerInterface(ctrl)
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("Got err when making")
		return
	}

	service := &UserHandler{
		UserRepo: mockRepo,
		Logger:   logger.Sugar(),
		Sessions: mockSessions,
	}

	tests := []struct {
		name        string
		setupMocks  func()
		authHeader  string
		wantStatus  int
		expectError bool
	}{
		{
			name: "Успешное получение пользователей",
			setupMocks: func() {
				mockSessions.EXPECT().Check(gomock.Any()).Return(&sessions.Session{})
				mockRepo.EXPECT().GetUsers().Return([]user.User{}, nil)
			},
			authHeader:  "Bearer validToken",
			wantStatus:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "Неверный токен авторизации",
			setupMocks:  func() {},
			authHeader:  "invalidToken",
			wantStatus:  http.StatusUnauthorized,
			expectError: true,
		},
		{
			name: "Ошибка при получении пользователей из репозитория",
			setupMocks: func() {
				mockSessions.EXPECT().Check(gomock.Any()).Return(&sessions.Session{})
				mockRepo.EXPECT().GetUsers().Return(nil, fmt.Errorf("database error"))
			},
			authHeader:  "Bearer validToken",
			wantStatus:  http.StatusBadRequest,
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks()

			req := httptest.NewRequest("GET", "/api/users", nil)
			req.Header.Add("Authorization", tc.authHeader)

			w := httptest.NewRecorder()

			service.GetUsers(w, req)

			resp := w.Result()
			assert.Equal(t, tc.wantStatus, resp.StatusCode)

			if tc.expectError {
				assert.NotEqual(t, http.StatusOK, resp.StatusCode)
			} else {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		})
	}
}

func TestSubscriptionHandlers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRepo := user.NewMockUserRepo(ctrl)
	mockSessions := sessions.NewMockSessionManagerInterface(ctrl)
	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Println("Got err when making")
		return
	}

	service := &UserHandler{
		UserRepo: mockRepo,
		Logger:   logger.Sugar(),
		Sessions: mockSessions,
	}

	tests := []struct {
		name        string
		handlerFunc func(http.ResponseWriter, *http.Request)
		setupMocks  func()
		subscribe   bool
		authHeader  string
		requestBody *SubscribeForm
		wantStatus  int
		expectError bool
	}{
		{
			name:        "Успешная подписка на пользователя",
			handlerFunc: service.SubscribeToUser,
			setupMocks: func() {
				mockSessions.EXPECT().Check(gomock.Any()).Return(&sessions.Session{})
				mockRepo.EXPECT().Subscribe(int64(2), int64(1), 1).Return(nil, nil)
			},
			subscribe:   true,
			authHeader:  "Bearer validToken",
			requestBody: &SubscribeForm{UserID: 2, SubscriberID: 1},
			wantStatus:  http.StatusOK,
			expectError: false,
		},
		{
			name:        "Успешная отписка от пользователя",
			handlerFunc: service.UnsubscribeToUser,
			setupMocks: func() {
				mockSessions.EXPECT().Check(gomock.Any()).Return(&sessions.Session{})
				mockRepo.EXPECT().Subscribe(int64(2), int64(1), 0).Return(nil, nil)
			},
			subscribe:   false,
			authHeader:  "Bearer validToken",
			requestBody: &SubscribeForm{UserID: 2, SubscriberID: 1},
			wantStatus:  http.StatusOK,
			expectError: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.setupMocks()

			body, err := json.Marshal(tc.requestBody)
			assert.NoError(t, err)

			req := httptest.NewRequest("POST", "/api/subscribe", bytes.NewReader(body))
			req.Header.Add("Authorization", tc.authHeader)

			w := httptest.NewRecorder()

			tc.handlerFunc(w, req)

			resp := w.Result()
			assert.Equal(t, tc.wantStatus, resp.StatusCode)

			if tc.expectError {
				assert.NotEqual(t, http.StatusOK, resp.StatusCode)
			} else {
				assert.Equal(t, http.StatusOK, resp.StatusCode)
			}
		})
	}
}
