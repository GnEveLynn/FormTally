package sms

import (
	"context"
	"sync"
)

type TestSender struct {
	mu    sync.RWMutex
	codes map[string]string
}

func NewTestSender() *TestSender { return &TestSender{codes: make(map[string]string)} }

func (s *TestSender) Send(_ context.Context, phone, purpose, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[phone+"\x00"+purpose] = code
	return nil
}

func (s *TestSender) LastCode(phone, purpose string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	code, ok := s.codes[phone+"\x00"+purpose]
	return code, ok
}
