package sms

import (
	"context"
	"log/slog"
	"sync"
)

type TestSender struct {
	mu     sync.RWMutex
	codes  map[string]string
	logger *slog.Logger
}

func NewTestSender(logger *slog.Logger) *TestSender {
	return &TestSender{codes: make(map[string]string), logger: logger}
}

func (s *TestSender) Send(_ context.Context, phone, purpose, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[phone+"\x00"+purpose] = code
	if s.logger != nil {
		s.logger.Info("test SMS code", "phone", phone, "purpose", purpose, "code", code)
	}
	return nil
}

func (s *TestSender) LastCode(phone, purpose string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	code, ok := s.codes[phone+"\x00"+purpose]
	return code, ok
}
