package systemsettings

import (
	"sync"
	"time"
)

type RefreshScheduler struct {
	mu       sync.Mutex
	interval time.Duration
	reset    chan struct{}
	stop     chan struct{}
	refresh  func()
}

func NewRefreshScheduler(interval time.Duration, refresh func()) *RefreshScheduler {
	s := &RefreshScheduler{interval: interval, reset: make(chan struct{}, 1), stop: make(chan struct{}), refresh: refresh}
	go s.loop()
	return s
}

func (s *RefreshScheduler) SetInterval(interval time.Duration) {
	s.mu.Lock()
	s.interval = interval
	s.mu.Unlock()
	select {
	case s.reset <- struct{}{}:
	default:
	}
}

func (s *RefreshScheduler) Close() { close(s.stop) }

func (s *RefreshScheduler) loop() {
	timer := time.NewTimer(s.getInterval())
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			s.refresh()
			timer.Reset(s.getInterval())
		case <-s.reset:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(s.getInterval())
		case <-s.stop:
			return
		}
	}
}

func (s *RefreshScheduler) getInterval() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.interval
}
