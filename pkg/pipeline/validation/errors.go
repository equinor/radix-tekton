package validation

import "github.com/pkg/errors"

var (
	ErrEmptyStepList             = errors.New("step list is empty")
	ErrSecretReferenceNotAllowed = errors.New("references to secrets are not allowed")
	ErrRadixVolumeNameNotAllowed = errors.New("volume name starting with radix are not allowed")
	ErrHostPathNotAllowed        = errors.New("HostPath is not allowed")
)
