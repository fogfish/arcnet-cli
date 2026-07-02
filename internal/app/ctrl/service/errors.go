package service

import "github.com/fogfish/faults"

const (
	ErrGitUnavailable     = faults.Type("git is required but was not found on PATH")
	ErrAlreadyInitialized = faults.Safe1[string]("%s is already an initialized graph")
	ErrTargetNotEmpty     = faults.Safe1[string]("%s is not empty; arc init requires an empty or non-existent directory")
	ErrLayoutWrite        = faults.Safe1[string]("failed to write graph layout at %s")
)
