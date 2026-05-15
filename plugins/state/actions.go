package state

import (
	"context"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/lmarqs/terraform-ui/pkg/sdk"
	"github.com/lmarqs/terraform-ui/pkg/sdk/frames"
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
								e.mutating = true
								return func() tea.Msg {
									err := svc.StateMove(context.Background(), address, dest)
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

func (e *Plugin) requestTaint(address string) tea.Cmd {
	svc := e.svc
	log := e.log
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(
				fmt.Sprintf("Taint %s? (will recreate on next apply)", address),
				func() tea.Cmd {
					e.mutating = true
					return func() tea.Msg {
						err := svc.Taint(context.Background(), address)
						if err != nil {
							log.Debug("state.taint.error", "address", address, "error", err.Error())
							return StateListMsg{Err: err}
						}
						log.Debug("state.taint.success", "address", address)
						return StateTaintedMsg{Addresses: []string{address}}
					}
				},
			),
		}
	}
}

func (e *Plugin) requestUntaint(address string) tea.Cmd {
	svc := e.svc
	log := e.log
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(
				fmt.Sprintf("Untaint %s?", address),
				func() tea.Cmd {
					e.mutating = true
					return func() tea.Msg {
						err := svc.Untaint(context.Background(), address)
						if err != nil {
							log.Debug("state.untaint.error", "address", address, "error", err.Error())
							return StateListMsg{Err: err}
						}
						log.Debug("state.untaint.success", "address", address)
						return StateUntaintedMsg{Addresses: []string{address}}
					}
				},
			),
		}
	}
}

func (e *Plugin) requestImport(address string) tea.Cmd {
	svc := e.svc
	log := e.log
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputText("Resource ID:", "", func(id string) tea.Cmd {
				if id == "" {
					return nil
				}
				return func() tea.Msg {
					return sdk.RequestInputMsg{
						Request: sdk.InputConfirm(
							fmt.Sprintf("Import %s as %s?", id, address),
							func() tea.Cmd {
								e.mutating = true
								return func() tea.Msg {
									err := svc.Import(context.Background(), address, id)
									if err != nil {
										log.Debug("state.import.error", "address", address, "id", id, "error", err.Error())
										return StateListMsg{Err: err}
									}
									log.Debug("state.import.success", "address", address, "id", id)
									return StateImportedMsg{Address: address, ID: id}
								}
							},
						),
					}
				}
			}),
		}
	}
}

// batchTaint taints multiple resources sequentially.
func (e *Plugin) batchTaint(addresses []string) tea.Cmd {
	svc := e.svc
	log := e.log
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(
				fmt.Sprintf("Taint %d resources? (will recreate on next apply)", len(addresses)),
				func() tea.Cmd {
					e.mutating = true
					return func() tea.Msg {
						for _, addr := range addresses {
							if err := svc.Taint(context.Background(), addr); err != nil {
								log.Debug("state.taint.error", "address", addr, "error", err.Error())
								return StateListMsg{Err: err}
							}
						}
						log.Debug("state.taint.batch.success", "count", len(addresses))
						return StateTaintedMsg{Addresses: addresses}
					}
				},
			),
		}
	}
}

// batchUntaint untaints multiple resources sequentially.
func (e *Plugin) batchUntaint(addresses []string) tea.Cmd {
	svc := e.svc
	log := e.log
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(
				fmt.Sprintf("Untaint %d resources?", len(addresses)),
				func() tea.Cmd {
					e.mutating = true
					return func() tea.Msg {
						for _, addr := range addresses {
							if err := svc.Untaint(context.Background(), addr); err != nil {
								log.Debug("state.untaint.error", "address", addr, "error", err.Error())
								return StateListMsg{Err: err}
							}
						}
						log.Debug("state.untaint.batch.success", "count", len(addresses))
						return StateUntaintedMsg{Addresses: addresses}
					}
				},
			),
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
					e.mutating = true
					return func() tea.Msg {
						for _, addr := range addresses {
							if err := svc.StateRm(context.Background(), addr); err != nil {
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
				if multiTarget {
					return e.batchTaint(targets)
				}
				return e.requestTaint(address)
			},
		},
		{
			Key:   "T",
			Label: "untaint",
			Handler: func() tea.Cmd {
				if multiTarget {
					return e.batchUntaint(targets)
				}
				return e.requestUntaint(address)
			},
		},
		{
			Key:      "n",
			Label:    "import",
			Disabled: multiTarget,
			Handler: func() tea.Cmd {
				return e.requestImport(address)
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
					e.mutating = true
					return func() tea.Msg {
						err := svc.StateRm(context.Background(), address)
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

func (e *Plugin) requestForceUnlock() tea.Cmd {
	lockID := e.lockInfo.ID
	return func() tea.Msg {
		return sdk.RequestInputMsg{
			Request: sdk.InputConfirm(
				fmt.Sprintf("Force-unlock %s? This is dangerous if another operation is running.", lockID),
				func() tea.Cmd {
					return func() tea.Msg { return ForceUnlockStartMsg{} }
				},
			),
		}
	}
}

func (e *Plugin) executeForceUnlock() tea.Cmd {
	lockID := e.lockInfo.ID
	svc := e.svc
	log := e.log
	return func() tea.Msg {
		err := svc.ForceUnlock(context.Background(), lockID)
		if err != nil {
			log.Debug("state.force-unlock.error", "lockID", lockID, "error", err.Error())
		} else {
			log.Debug("state.force-unlock.success", "lockID", lockID)
		}
		return ForceUnlockResultMsg{Err: err}
	}
}
