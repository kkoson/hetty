// Package intercept provides request/response interception functionality
// for the Hetty proxy, allowing inspection and modification of HTTP traffic.
package intercept

import (
	"context"
	"errors"
	"net/http"
	"sync"
)

// ErrRequestDropped is returned when an intercepted request is dropped by the user.
var ErrRequestDropped = errors.New("intercept: request dropped")

// ErrResponseDropped is returned when an intercepted response is dropped by the user.
var ErrResponseDropped = errors.New("intercept: response dropped")

// Request represents an intercepted HTTP request awaiting a decision.
type Request struct {
	ID      string
	Req     *http.Request
	respCh  chan Response
	errCh   chan error
}

// Response represents the result of a reviewed intercepted request.
type Response struct {
	Req  *http.Request
	Drop bool
}

// Service manages interception of HTTP requests and responses.
type Service struct {
	mu       sync.RWMutex
	enabled  bool
	pending  map[string]*Request
	requestCh chan *Request
}

// NewService creates a new interception Service.
func NewService() *Service {
	return &Service{
		pending:   make(map[string]*Request),
		requestCh: make(chan *Request, 100),
	}
}

// SetEnabled enables or disables request interception.
func (s *Service) SetEnabled(enabled bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.enabled = enabled
}

// IsEnabled returns whether interception is currently enabled.
func (s *Service) IsEnabled() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.enabled
}

// InterceptRequest intercepts an HTTP request if interception is enabled.
// It blocks until the request is reviewed or the context is cancelled.
func (s *Service) InterceptRequest(ctx context.Context, id string, req *http.Request) (*http.Request, error) {
	if !s.IsEnabled() {
		return req, nil
	}

	pendingReq := &Request{
		ID:     id,
		Req:    req,
		respCh: make(chan Response, 1),
		errCh:  make(chan error, 1),
	}

	s.mu.Lock()
	s.pending[id] = pendingReq
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pending, id)
		s.mu.Unlock()
	}()

	select {
	case s.requestCh <- pendingReq:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	select {
	case resp := <-pendingReq.respCh:
		if resp.Drop {
			return nil, ErrRequestDropped
		}
		return resp.Req, nil
	case err := <-pendingReq.errCh:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ReviewRequest submits a review decision for a pending intercepted request.
func (s *Service) ReviewRequest(id string, req *http.Request, drop bool) error {
	s.mu.RLock()
	pendingReq, ok := s.pending[id]
	s.mu.RUnlock()

	if !ok {
		return errors.New("intercept: request not found")
	}

	pendingReq.respCh <- Response{
		Req:  req,
		Drop: drop,
	}

	return nil
}

// Requests returns a channel that receives intercepted requests for review.
func (s *Service) Requests() <-chan *Request {
	return s.requestCh
}

// PendingRequests returns a snapshot of all currently pending intercepted requests.
func (s *Service) PendingRequests() []*Request {
	s.mu.RLock()
	defer s.mu.RUnlock()

	reqs := make([]*Request, 0, len(s.pending))
	for _, r := range s.pending {
		reqs = append(reqs, r)
	}
	return reqs
}
