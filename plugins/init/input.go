package init

// Input is the typed input port for the init plugin.
type Input struct {
	Upgrade       bool
	Reconfigure   bool
	Backend       *bool
	BackendConfig []string
}
