package sdk

// Status represents the common lifecycle states for data-fetching plugins.
// Plugins can extend with their own constants starting from offset 10:
//
//	const StatusShowingDetail = sdk.Status(10)
type Status int

const (
	StatusIdle    Status = iota // not yet activated
	StatusLoading               // async operation in progress
	StatusDone                  // data loaded successfully
	StatusError                 // operation failed
)

func (s Status) IsIdle() bool    { return s == StatusIdle }
func (s Status) IsLoading() bool { return s == StatusLoading }
func (s Status) IsReady() bool   { return s == StatusDone }
func (s Status) HasError() bool  { return s == StatusError }
