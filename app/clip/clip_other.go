//go:build !windows && !darwin

package clip

// Only the Windows and macOS clipboards are read for now.
func read() ([]Image, error) { return nil, ErrNone }
