package pipe

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestError(t *testing.T) {
	var err error
	var e Error

	t.Run("error", func(t *testing.T) {
		err = NewError(42, errors.New("pipe.Error"))
		e = AsError(err)
		require.EqualValues(t, 42, e.Code)
		require.EqualError(t, e.Err, "pipe.Error")
		require.EqualError(t, err, "Error{Code: 42, Err: pipe.Error}")
	})

	t.Run("errorf", func(t *testing.T) {
		err = NewErrorf(142, "pipe: %w", errors.New("Errorf"))
		e = AsError(err)
		require.EqualValues(t, 142, e.Code)
		require.EqualError(t, e.Err, "pipe: Errorf")
		require.EqualError(t, err, "Error{Code: 142, Err: pipe: Errorf}")
	})

	t.Run("as error", func(t *testing.T) {
		err = errors.New("random error")
		e = AsError(err)
		require.EqualValues(t, UnknownError, e.Code)
		require.EqualError(t, e.Err, "random error")
		require.EqualError(t, e, "Error{Code: 250, Err: random error}")
	})
}
