package sessions

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"

	"github.com/gomodule/redigo/redis"
)

type Session struct {
	ID        int64
	Login     string
	Useragent string
}

type SessionID struct {
	ID string
}

const sessKeyLen = 10

type SessionManagerInterface interface {
	Create(in *Session) (*SessionID, error)
	Check(in *SessionID) *Session
}

type SessionManager struct {
	redisConn redis.Conn
}

func NewSessionManager(conn redis.Conn) *SessionManager {
	return &SessionManager{
		redisConn: conn,
	}
}

func (sm *SessionManager) Create(in *Session) (*SessionID, error) {
	id := SessionID{RandStringRunes(sessKeyLen)}
	dataSerialized, err := json.Marshal(in)
	if err != nil {
		return nil, fmt.Errorf("can't marshal data")
	}
	mkey := "sessions:" + id.ID
	result, err := redis.String(sm.redisConn.Do("SET", mkey, dataSerialized, "EX", 3600))
	if err != nil {
		return nil, err
	}
	if result != "OK" {
		return nil, fmt.Errorf("result not OK")
	}
	return &id, nil
}

func (sm *SessionManager) Check(in *SessionID) *Session {
	mkey := "sessions:" + in.ID
	data, err := redis.Bytes(sm.redisConn.Do("GET", mkey))
	if err != nil {
		log.Println("cant get data:", err)
		return nil
	}
	sess := &Session{}
	err = json.Unmarshal(data, sess)
	if err != nil {
		log.Println("cant unpack session data:", err)
		return nil
	}
	return sess
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
