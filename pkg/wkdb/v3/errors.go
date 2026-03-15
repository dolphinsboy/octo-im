package v3

import (
	"errors"
	"fmt"
)

var ErrNotImplemented = errors.New("not implemented")

func errNotImplemented(op string) error {
	return fmt.Errorf("%s: %w", op, ErrNotImplemented)
}
