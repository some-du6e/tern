//go:build !windows

package t3

// Elsewhere the files rely on 0600 permissions in a 0700 directory. A
// Keychain or Secret Service backend can replace this later.
func protect(data []byte) ([]byte, error)   { return data, nil }
func unprotect(data []byte) ([]byte, error) { return data, nil }
