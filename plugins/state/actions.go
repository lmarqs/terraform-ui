package state

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
	tfuiimport "github.com/lmarqs/terraform-ui/plugins/import"
	"github.com/lmarqs/terraform-ui/plugins/taint"
	"github.com/lmarqs/terraform-ui/plugins/untaint"
)

func (e *Plugin) requestMove(address string) tea.Cmd {
	svc := e.svc
	log := e.log
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputText("Move to:", address, func(dest string) tea.Cmd {
				if dest == "" || dest == address {
					return nil
				}
				return func() tea.Msg {
					return sdk.RequestInputMsg{
						Request: sdk.InputConfirm(
							fmt.Sprintf("Move %s → %s?", address, dest),
							func() tea.Cmd {
								e.Cancel()
								ctx, cancel := context.WithCancel(context.Background())
								e.cancelFn = cancel
								e.mutating = true
								return func() tea.Msg {
									err := svc.StateMove(ctx, address, dest)
									if err != nil {
										log.Debug("state.move.error", "source", address, "dest", dest, "error", err.Error())
										return StateListMsg{Err: err}
									}
									log.Debug("state.move.success", "source", address, "dest", dest)
									return StateMovedMsg{Source: address, Dest: dest}
								}
							},
						),
					}
				}
			}),
		}
	}
}

// batchDelete deletes multiple resources sequentially.
func (e *Plugin) batchDelete(addresses []string) tea.Cmd {
	svc := e.svc
	log := e.log
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(
				fmt.Sprintf("Remove %d resources from state?", len(addresses)),
				func() tea.Cmd {
					e.Cancel()
					ctx, cancel := context.WithCancel(context.Background())
					e.cancelFn = cancel
					e.mutating = true
					return func() tea.Msg {
						for _, addr := range addresses {
							if err := svc.StateRm(ctx, addr); err != nil {
								log.Debug("state.rm.error", "address", addr, "error", err.Error())
								return StateListMsg{Err: err}
							}
						}
						log.Debug("state.rm.batch.success", "count", len(addresses))
						return StateDeletedMsg{Address: fmt.Sprintf("%d resources", len(addresses))}
					}
				},
			),
		}
	}
}

// actionTargets returns the addresses to act on: pinned if any, otherwise cursor.
func (e *Plugin) actionTargets() []string {
	if e.pins != nil && e.pins.Count() > 0 {
		return e.pins.All()
	}
	r := e.SelectedResource()
	if r.Address != "" {
		return []string{r.Address}
	}
	return nil
}

// buildActionFrame creates the action palette for the given address context.
func (e *Plugin) buildActionFrame(address string, batch bool) *frames.ActionFrame {
	pinCount := 0
	if e.pins != nil {
		pinCount = e.pins.Count()
	}
	multiTarget := batch && pinCount > 1

	title := address
	if multiTarget {
		title = fmt.Sprintf("%d pinned resources", pinCount)
	}

	targets := e.actionTargets()

	actions := []frames.Action{
		{
			Key:   "d",
			Label: "delete",
			Handler: func() tea.Cmd {
				if multiTarget {
					return e.batchDelete(targets)
				}
				return e.requestDelete(address)
			},
		},
		{
			Key:      "m",
			Label:    "move",
			Disabled: multiTarget,
			Handler: func() tea.Cmd {
				return e.requestMove(address)
			},
		},
		{
			Key:   "t",
			Label: "taint",
			Handler: func() tea.Cmd {
				return func() tea.Msg { return taint.TaintRequestMsg{Addresses: targets} }
			},
		},
		{
			Key:   "T",
			Label: "untaint",
			Handler: func() tea.Cmd {
				return func() tea.Msg { return untaint.UntaintRequestMsg{Addresses: targets} }
			},
		},
		{
			Key:      "n",
			Label:    "import",
			Disabled: multiTarget,
			Handler: func() tea.Cmd {
				return func() tea.Msg { return tfuiimport.ImportRequestMsg{Address: address} }
			},
		},
		{
			Key:   "e",
			Label: "edit",
			Handler: func() tea.Cmd {
				if multiTarget {
					return e.requestEditMultiple(targets)
				}
				return e.requestEdit(address)
			},
		},
	}

	return frames.NewActionFrame(title, actions)
}

func (e *Plugin) requestDelete(address string) tea.Cmd {
	svc := e.svc
	log := e.log
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(
				fmt.Sprintf("Remove %s from state?", address),
				func() tea.Cmd {
					e.Cancel()
					ctx, cancel := context.WithCancel(context.Background())
					e.cancelFn = cancel
					e.mutating = true
					return func() tea.Msg {
						err := svc.StateRm(ctx, address)
						if err != nil {
							log.Debug("state.rm.error", "address", address, "error", err.Error())
							return StateListMsg{Err: err}
						}
						log.Debug("state.rm.success", "address", address)
						return StateDeletedMsg{Address: address}
					}
				},
			),
		}
	}
}

func (e *Plugin) requestEdit(address string) tea.Cmd {
	return func() tea.Msg {
		return StateEditMsg{Address: address}
	}
}

func (e *Plugin) requestEditMultiple(addresses []string) tea.Cmd {
	return func() tea.Msg {
		return StateEditMsg{Addresses: addresses}
	}
}
