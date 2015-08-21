package testutils

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"
)

type WriteVerifier interface {
	io.Writer
	CheckDone()
}

type parallelWriteVerifier struct {
	name           string
	expectedWrites []string
	writeMatch     []bool
	t              *testing.T
	lock           *sync.RWMutex
}

func NewParallelWV(name string, expectedWrites []string, t *testing.T) WriteVerifier {
	return &parallelWriteVerifier{
		name:           name,
		expectedWrites: expectedWrites,
		writeMatch:     make([]bool, len(expectedWrites)),
		t:              t,
		lock:           &sync.RWMutex{},
	}
}

func (v *parallelWriteVerifier) CheckDone() {
	done := true
	for _, m := range v.writeMatch {
		if m == false {
			done = false
		}
	}
	if !done {
		v.t.Errorf("%s writer did not receive all expected writes.\n%#v\n%#v", v.name, v.expectedWrites, v.writeMatch)
	}
}

func (v *parallelWriteVerifier) Write(p []byte) (int, error) {
	v.lock.Lock()
	defer v.lock.Unlock()

	for i, e := range v.expectedWrites {
		if e == string(p) && v.writeMatch[i] == false {
			v.writeMatch[i] = true

			if string(p) == "Fail this write" {
				return -1, errors.New("I was told to fail this!")
			}
			return len(p), nil
		}
	}
	v.t.Errorf("%s writer received unexpected write: \"%s\"\n%#v\n%#v", v.name, p, v.expectedWrites, v.writeMatch)
	return len(p), nil
}

type sequentialWriteVerifier struct {
	name           string
	expectedWrites []string
	currentIndex   int
	t              *testing.T
}

func NewSequentialWV(name string, expectedWrites []string, t *testing.T) WriteVerifier {
	return &sequentialWriteVerifier{
		name:           name,
		expectedWrites: expectedWrites,
		currentIndex:   0,
		t:              t,
	}
}

func (s *sequentialWriteVerifier) CheckDone() {
	if s.currentIndex != len(s.expectedWrites) {
		s.t.Errorf("%s writer did not receive all expected writes.", s.name)
	}
}

func (s *sequentialWriteVerifier) Write(p []byte) (int, error) {
	if s.currentIndex >= len(s.expectedWrites) {
		s.t.Errorf("%s writer was not expecting a write but received \"%s\"", s.name, p)
		return len(p), nil
	}
	if !bytes.Equal(p, []byte(s.expectedWrites[s.currentIndex])) {
		s.t.Errorf("%s writer at index %d expected \"%s\" but received \"%s\"", s.name, s.currentIndex, s.expectedWrites[s.currentIndex], p)
	}
	s.currentIndex++
	if string(p) == "Fail this write" {
		return -1, errors.New("I was told to fail this!")
	}
	return len(p), nil
}
