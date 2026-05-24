package sdk

// Resource represents a terraform-managed resource identified by its address,
// type, logical name, module path, and provider.
type Resource struct {
	Address      string
	Type         string
	Name         string
	Module       string
	ProviderName string
	Tainted      bool
}
