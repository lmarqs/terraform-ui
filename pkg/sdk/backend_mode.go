package sdk

// BackendMode encodes the terraform init backend strategy.
type BackendMode int

const (
	BackendDefault  BackendMode = iota // omit — terraform decides (init with backend)
	BackendEnabled                     // -backend=true
	BackendDisabled                    // -backend=false
)
