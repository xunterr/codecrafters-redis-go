package storage

import (
	"errors"
	"log"
	"sync"
	"time"
)

type Storage struct {
	mu      sync.RWMutex
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
		s.mu.Lock()
		delete(s.storage, key)
		s.mu.Unlock()
	}(expire, key)
	return nil
}

func (s *Storage) Set(key string, value string) error {
	log.Printf("SET: %s - %s", key, value)

	s.mu.RLock()
	_, ok := s.storage[key]
	s.mu.RUnlock()
	if ok {
		return errors.New("Key already exists")
	}

	s.mu.Lock()
	s.storage[key] = value
	s.mu.Unlock()
	return nil
}

func (s *Storage) Get(key string) (string, error) {
	log.Printf("GET: %s", key)

	s.mu.RLock()
	value, ok := s.storage[key]
	s.mu.RUnlock()
	if !ok {
		log.Printf("%v", s.storage)
		return "", errors.New("No such key")
	}
	return value, nil
}
