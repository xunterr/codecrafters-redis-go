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
	log.Printf("SET: %s - %s", key, value)
	if _, ok := s.storage[key]; ok {
		log.Printf("%v", s.storage)
		return errors.New("Key already exists")
	}
	s.storage[key] = value
	return nil
}

func (s *Storage) Get(key string) (string, error) {
	log.Printf("GET: %s", key)
	value, ok := s.storage[key]
	if !ok {
		log.Printf("%v", s.storage)
		return "", errors.New("No such key")
	}
	return value, nil
}
