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

// LockTimeout is a terraform duration for state lock acquisition.
// Empty string means terraform default.
type LockTimeout string

func (lt LockTimeout) String() string { return string(lt) }
