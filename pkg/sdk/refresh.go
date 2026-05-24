package sdk

// RefreshMode encodes the terraform refresh strategy for plan operations.
type RefreshMode int

const (
	RefreshDefault  RefreshMode = iota // omit — terraform decides
	RefreshEnabled                     // -refresh=true
	RefreshDisabled                    // -refresh=false
	RefreshOnly                        // -refresh-only
)
