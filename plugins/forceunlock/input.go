package forceunlock

// Input is the typed input port for the force-unlock plugin.
type Input struct {
	LockID string
	Force  bool
}
