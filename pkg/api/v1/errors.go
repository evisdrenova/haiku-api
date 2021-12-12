package v1

import "errors"

type ErrorReason string

const (
	reasonAlreadyExists = "entity already exists"
)

var (
	ErrAlreadyExists = errors.New(reasonAlreadyExists)
)

func IsAlreadyExists(err error) bool {
	return errors.Is(err, ErrAlreadyExists)
}
