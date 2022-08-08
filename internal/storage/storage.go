package storage

import (
	"errors"
	"strconv"
)

type storage struct {
	values map[string]string
}

func NewStorage() *storage {
	return &storage{
		values: make(map[string]string),
	}
}

var ErrKeyNotFound = errors.New("key not found")

func (s *storage) SetValue(key, value string) error {
	s.values[key] = value
	return nil
}

func (s *storage) GetValue(key string) (string, error) {
	value, ok := s.values[key]
	if ok == false {
		return "", ErrKeyNotFound
	}
	return value, nil
}

func (s *storage) GetOrDefaultValue(key, defaultValue string) (string, error) {
	value, err := s.GetValue(key)
	if err == ErrKeyNotFound {
		return defaultValue, nil
	}
	return value, err
}

func (s *storage) SetInt64Value(key string, value int64) error {
	return s.SetValue(key, strconv.FormatInt(value, 10))
}

func (s *storage) GetInt64Value(key string) (int64, error) {
	value, err := s.GetValue(key)
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(value, 10, 64)
}
