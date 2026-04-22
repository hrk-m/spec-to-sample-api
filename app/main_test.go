package main

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubDBPinger struct {
	errs       []error
	defaultErr error
	calls      int
}

func (s *stubDBPinger) PingContext(_ context.Context) error {
	s.calls++

	if len(s.errs) == 0 {
		return s.defaultErr
	}

	err := s.errs[0]
	s.errs = s.errs[1:]

	return err
}

func TestWaitForMySQL_RetriesUntilReady(t *testing.T) {
	t.Parallel()

	db := &stubDBPinger{
		errs: []error{
			errors.New("unexpected EOF"),
			errors.New("invalid connection"),
			nil,
		},
	}

	err := waitForMySQL(context.Background(), db, time.Millisecond, log.New(io.Discard, "", 0))

	require.NoError(t, err)
	assert.Equal(t, 3, db.calls)
}

func TestWaitForMySQL_StopsAtDeadline(t *testing.T) {
	t.Parallel()

	db := &stubDBPinger{
		errs: []error{
			errors.New("unexpected EOF"),
			errors.New("invalid connection"),
			errors.New("still starting"),
		},
		defaultErr: errors.New("still starting"),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err := waitForMySQL(ctx, db, time.Millisecond, log.New(io.Discard, "", 0))

	require.Error(t, err)
	assert.ErrorContains(t, err, "timed out waiting for MySQL")
	assert.ErrorContains(t, err, "still starting")
}
