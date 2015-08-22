package testutils

import (
	"bytes"
	"errors"
	"sync"
	"testing"
)

const (
	// FailWrite is the string that should be sent to a WriteVerfier to test
	// Write failures. This string will cause Write calls to return -1,
	// errors.New("I was told to fail this!")
	FailWrite = "Fail this write!"
)

// WriteVerifier functions as an io.Writer allowing users to test code that should
// write to an io.Writer interface. It calls it's testing.T instance's Error
// on unexpected writes and then the user can call CheckDone to verify all expected
// writes did indeed take place
type WriteVerifier interface {
	// Write emulates the Write method of io.Writers but instead compares it
	// against an internal set of expected writes. If the write does not match an
	// expected write that has yet to take place it will call testing.T.Error
	Write(p []byte) (n int, err error)

	// CheckDone should be called after all writes to the write should have
	// completed. The object then validates that all expected writes were received,
	// calling the Error method in the testing.T instance if any were missed.
	CheckDone()
}

type parallelWriteVerifier struct {
	name           string
	expectedWrites []string
	writeMatch     []bool
	t              *testing.T
	lock           *sync.RWMutex
}

// NewParallelWV creates a concurrency enabled WriteVerifier for testing code that
// concurrently writes to io.Writers
func NewParallelWV(name string, expectedWrites []string, t *testing.T) WriteVerifier {
	return &parallelWriteVerifier{
		name:           name,
		expectedWrites: expectedWrites,
		writeMatch:     make([]bool, len(expectedWrites)),
		t:              t,
		lock:           &sync.RWMutex{},
	}
}

// CheckDone should be called after all writes to the write should have
// completed. The object then validates that all expected writes were received,
// calling the Error method in the testing.T instance if any were missed.
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

// Write emulates the Write method of io.Writers but instead compares it
// against an internal set of expected writes. If the write does not match an
// expected write that has yet to take place it will call testing.T.Error
func (v *parallelWriteVerifier) Write(p []byte) (int, error) {
	v.lock.Lock()
	defer v.lock.Unlock()

	for i, e := range v.expectedWrites {
		if e == string(p) && v.writeMatch[i] == false {
			v.writeMatch[i] = true

			if bytes.Equal(p, []byte(FailWrite)) {
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

// NewSequentialWV creates a sequential WriteVerifier which can be used to test
// code writing to an io.Writer interface non-concurrently
func NewSequentialWV(name string, expectedWrites []string, t *testing.T) WriteVerifier {
	return &sequentialWriteVerifier{
		name:           name,
		expectedWrites: expectedWrites,
		currentIndex:   0,
		t:              t,
	}
}

// CheckDone should be called after all writes to the write should have
// completed. The object then validates that all expected writes were received,
// calling the Error method in the testing.T instance if any were missed.
func (s *sequentialWriteVerifier) CheckDone() {
	if s.currentIndex != len(s.expectedWrites) {
		s.t.Errorf("%s writer did not receive all expected writes.", s.name)
	}
}

// Write emulates the Write method of io.Writers but instead compares it
// against an internal set of expected writes. If the write does not match an
// expected write that has yet to take place it will call testing.T.Error
func (s *sequentialWriteVerifier) Write(p []byte) (int, error) {
	if s.currentIndex >= len(s.expectedWrites) {
		s.t.Errorf("%s writer was not expecting a write but received \"%s\"", s.name, p)
		return len(p), nil
	}
	if !bytes.Equal(p, []byte(s.expectedWrites[s.currentIndex])) {
		s.t.Errorf("%s writer at index %d expected \"%s\" but received \"%s\"", s.name, s.currentIndex, s.expectedWrites[s.currentIndex], p)
	}
	s.currentIndex++
	if bytes.Equal(p, []byte(FailWrite)) {
		return -1, errors.New("I was told to fail this!")
	}
	return len(p), nil
}
