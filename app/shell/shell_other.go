//go:build !windows && !darwin

package shell

import "errors"

var kinds = map[string]bool{}

func run(kind, code string) error { return errors.New("running code needs Windows or a Mac") }
