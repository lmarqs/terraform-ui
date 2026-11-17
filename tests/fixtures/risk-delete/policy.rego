package tfui.risk.v1

import rego.v1

classification := {"level": "high", "reasons": [{"code": "delete-after-absent", "message": "Deletion with an absent after value.", "paths": []}]} if {
	input.change.actions == ["delete"]
	input.change.after.kind == "absent"
}

classification := {"level": "low", "reasons": [{"code": "delete-after-present", "message": "Deletion with a present after value.", "paths": []}]} if {
	input.change.actions == ["delete"]
	input.change.after.kind != "absent"
}
