package sdk

// LockMode encodes the terraform state-locking strategy.
type LockMode int

const (
	LockDefault  LockMode = iota // omit — terraform decides
	LockEnabled                  // -lock=true
	LockDisabled                 // -lock=false
)

// LockModeFromPtr converts a legacy *bool to LockMode.
func LockModeFromPtr(p *bool) LockMode {
	if p == nil {
		return LockDefault
	}
	if *p {
		return LockEnabled
	}
	return LockDisabled
}

// Or returns the receiver when it names an explicit locking choice, and the
// fallback when it does not. Lets a per-invocation flag override a resolved
// config value without a caller-side Changed() check at every use site.
func (m LockMode) Or(fallback LockMode) LockMode {
	if m == LockDefault {
		return fallback
	}
	return m
}

// LockTimeout is a terraform duration for state lock acquisition.
// Empty string means terraform default.
type LockTimeout string

func (lt LockTimeout) String() string { return string(lt) }

// Or returns the receiver when set, and the fallback when empty.
func (lt LockTimeout) Or(fallback LockTimeout) LockTimeout {
	if lt == "" {
		return fallback
	}
	return lt
}
