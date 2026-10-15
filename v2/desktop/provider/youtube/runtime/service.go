package runtimeenv

import (
	"context"
	"strings"
	"sync"
)

type Service struct {
	mu            sync.RWMutex
	sharedRuntime string
	status        Status
}

func NewService(sharedRuntime string) *Service {
	service := &Service{
		sharedRuntime: strings.TrimSpace(sharedRuntime),
	}
	service.Refresh()
	return service
}

func (s *Service) Refresh() Status {
	if s == nil {
		return Status{}
	}
	status := Resolve(s.sharedRuntime)
	s.mu.Lock()
	s.status = status
	s.mu.Unlock()
	return status
}

func (s *Service) Status() Status {
	if s == nil {
		return Status{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

func (s *Service) Python() string {
	return s.Status().Python
}

func (s *Service) Setup(ctx context.Context) (string, error) {
	if s == nil {
		return "", context.Canceled
	}
	message, err := Setup(ctx, s.sharedRuntime)
	if err != nil {
		return "", err
	}
	s.Refresh()
	return message, nil
}
