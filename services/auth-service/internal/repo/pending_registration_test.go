package repo

import (
	"errors"
	"testing"
)

func TestErrPendingRegistrationActiveSentinel(t *testing.T) {
	wrapped := errors.Join(errors.New("ctx"), ErrPendingRegistrationActive)
	if !errors.Is(wrapped, ErrPendingRegistrationActive) {
		t.Fatal("ErrPendingRegistrationActive must be detectable with errors.Is")
	}
}
