package izero

import "errors"

var (
	InvalidData       = errors.New("Invalid data")
	ResizeFailed      = errors.New("Resize failed")
	InvalidDimensions = errors.New("Invalid dimensions")
)
