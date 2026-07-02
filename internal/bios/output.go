package bios

import "encoding/json"

// Mode is the single resolved output mode for the current invocation.
type Mode int

const (
	ModeHuman Mode = iota
	ModeVerbose
	ModeSilent
	ModeJSON
)

// DS-03 persistent flags, bound directly from cmd/arc/root.go so that
// ResolveMode stays the only place flag-to-mode priority is decided.
var (
	Quiet   bool
	Verbose bool
	JSON    bool
	Color   bool
)

// ResolveMode is the ONLY place flag-to-mode priority is decided. Every
// command calls this; no command re-derives the priority order.
func ResolveMode() Mode {
	switch {
	case JSON:
		return ModeJSON
	case Verbose:
		return ModeVerbose
	default:
		return ModeHuman
	}
}

// Printer renders a domain value T to output bytes for exactly one output
// mode.
type Printer[T any] interface {
	Show(T) ([]byte, error)
}

// Registry binds a domain type T's bespoke human-mode renderers. JSON and
// Silent need no entry — the registry supplies them for every T
// automatically.
type Registry[T any] struct {
	Human   Printer[T]
	Verbose Printer[T]
}

func (r Registry[T]) Resolve(mode Mode) Printer[T] {
	switch mode {
	case ModeJSON:
		return jsonPrinter[T]{}
	case ModeSilent:
		return nonePrinter[T]{}
	case ModeVerbose:
		if r.Verbose != nil {
			return r.Verbose
		}
		return r.Human
	default:
		return r.Human
	}
}

type jsonPrinter[T any] struct{}

func (jsonPrinter[T]) Show(v T) ([]byte, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

type nonePrinter[T any] struct{}

func (nonePrinter[T]) Show(T) ([]byte, error) { return nil, nil }
