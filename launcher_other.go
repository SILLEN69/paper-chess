//go:build !linux

package main

import "errors"

func monotonicMicros() (int64, error) { return 0, errors.New("tablet clock requires Linux") }

func watchButtons() error          { return errors.New("power shortcut requires tablet Linux") }
func diagnosticTap(x, y int) error { return errors.New("touch diagnostics require tablet Linux") }
