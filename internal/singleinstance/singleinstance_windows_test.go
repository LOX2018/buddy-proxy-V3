//go:build windows

package singleinstance

import (
	"errors"
	"testing"
)

func TestAcquireBlocksSecond(t *testing.T) {
	const name = "CodeBuddyProxySingletonUnitTest"
	m1, err := Acquire(name)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	defer m1.Release()

	_, err = Acquire(name)
	if !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("second acquire: got err=%v, want ErrAlreadyRunning", err)
	}
}

func TestReleaseAllowsReacquire(t *testing.T) {
	const name = "CodeBuddyProxySingletonUnitTestRelease"
	m, err := Acquire(name)
	if err != nil {
		t.Fatalf("acquire failed: %v", err)
	}
	m.Release()

	m2, err := Acquire(name)
	if err != nil {
		t.Fatalf("re-acquire after release failed: %v", err)
	}
	m2.Release()
}

func TestDifferentNamesIndependent(t *testing.T) {
	m1, err := Acquire("UnitTestNameA")
	if err != nil {
		t.Fatalf("acquire A: %v", err)
	}
	defer m1.Release()

	m2, err := Acquire("UnitTestNameB")
	if err != nil {
		t.Fatalf("acquire B while A held: %v", err)
	}
	m2.Release()
}
