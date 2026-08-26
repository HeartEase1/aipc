package service

import (
	"sync"
	"time"
)

const (
	openAIOAuth429TransientStreakTTL      = 10 * time.Minute
	openAIOAuth429TransientFirstCooldown  = 5 * time.Second
	openAIOAuth429TransientSecondCooldown = 15 * time.Second
	openAIOAuth429TransientLongCooldown   = time.Minute
	openAIOAuth429TransientDefaultMax     = 4096
)

type openAIOAuth429TransientEntry struct {
	failureStreak int
	lastFailure   time.Time
	blockUntil    time.Time
	lastTouched   time.Time
}

type openAIOAuth429TransientDecision struct {
	FailureStreak int
	Cooldown      time.Duration
	BlockUntil    time.Time
}

type openAIOAuth429TransientState struct {
	mu         sync.Mutex
	entries    map[int64]openAIOAuth429TransientEntry
	maxEntries int
}

func newOpenAIOAuth429TransientState(maxEntries int) *openAIOAuth429TransientState {
	if maxEntries <= 0 {
		maxEntries = openAIOAuth429TransientDefaultMax
	}
	return &openAIOAuth429TransientState{
		entries:    make(map[int64]openAIOAuth429TransientEntry),
		maxEntries: maxEntries,
	}
}

func (s *openAIOAuth429TransientState) recordFailure(accountID int64, now time.Time) openAIOAuth429TransientDecision {
	if s == nil || accountID <= 0 {
		return openAIOAuth429TransientDecision{}
	}
	if now.IsZero() {
		now = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.entries == nil {
		s.entries = make(map[int64]openAIOAuth429TransientEntry)
	}
	if s.maxEntries <= 0 {
		s.maxEntries = openAIOAuth429TransientDefaultMax
	}

	entry, exists := s.entries[accountID]
	if !exists {
		s.evictOldestLocked()
	}
	if !exists || entry.lastFailure.IsZero() || now.Before(entry.lastFailure) || now.Sub(entry.lastFailure) > openAIOAuth429TransientStreakTTL {
		entry = openAIOAuth429TransientEntry{}
	}

	entry.failureStreak++
	entry.lastFailure = now
	entry.lastTouched = now
	entry.blockUntil = now.Add(openAIOAuth429TransientCooldown(entry.failureStreak))
	s.entries[accountID] = entry

	return openAIOAuth429TransientDecision{
		FailureStreak: entry.failureStreak,
		Cooldown:      entry.blockUntil.Sub(now),
		BlockUntil:    entry.blockUntil,
	}
}

func openAIOAuth429TransientCooldown(failureStreak int) time.Duration {
	switch {
	case failureStreak >= 3:
		return openAIOAuth429TransientLongCooldown
	case failureStreak == 2:
		return openAIOAuth429TransientSecondCooldown
	default:
		return openAIOAuth429TransientFirstCooldown
	}
}

func (s *openAIOAuth429TransientState) recordSuccess(accountID int64) {
	if s == nil || accountID <= 0 {
		return
	}
	s.mu.Lock()
	delete(s.entries, accountID)
	s.mu.Unlock()
}

func (s *openAIOAuth429TransientState) isBlocked(accountID int64, now time.Time) bool {
	if s == nil || accountID <= 0 {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	entry, exists := s.entries[accountID]
	if !exists {
		return false
	}
	if now.Before(entry.lastFailure) || now.Sub(entry.lastFailure) > openAIOAuth429TransientStreakTTL {
		delete(s.entries, accountID)
		return false
	}
	entry.lastTouched = now
	s.entries[accountID] = entry
	return now.Before(entry.blockUntil)
}

func (s *openAIOAuth429TransientState) size() int {
	if s == nil {
		return 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

func (s *openAIOAuth429TransientState) evictOldestLocked() {
	if len(s.entries) < s.maxEntries {
		return
	}
	var oldestAccountID int64
	var oldestTime time.Time
	found := false
	for accountID, entry := range s.entries {
		if !found || entry.lastTouched.Before(oldestTime) {
			oldestAccountID = accountID
			oldestTime = entry.lastTouched
			found = true
		}
	}
	if found {
		delete(s.entries, oldestAccountID)
	}
}
