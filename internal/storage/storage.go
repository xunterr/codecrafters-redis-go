package storage

import (
	"errors"
	"log"
	"time"
)

type Storage struct {
	storage map[string]string
}

func NewStorage() *Storage {
	return &Storage{
		storage: make(map[string]string),
	}
}

func (s *Storage) SetWithTimer(key string, value string, expire int) error {
	err := s.Set(key, value)
	if err != nil {
		return err
	}

	go func(expire int, key string) {
		expiryMs := time.Millisecond * time.Duration(expire)
		timer := time.After(expiryMs)
		log.Printf("Expiry (ms): %d", expire)
		<-timer
		delete(s.storage, key)
	}(expire, key)
	return nil
}

func (s *Storage) Set(key string, value string) error {
	if _, ok := s.storage[key]; ok {
		return errors.New("Key already exists")
	}
	s.storage[key] = value
	return nil
}

func (s *Storage) Get(key string) (string, error) {
	value, ok := s.storage[key]
	if !ok {
		return "", errors.New("No such key")
	}
	return value, nil
}
