package tfui.risk.v1

import rego.v1

classification := {"level": "low", "reasons": [{"code": "current-object", "message": "Current object update.", "paths": []}]} if {
	input.resource.deposed == null
}

classification := {"level": "high", "reasons": [{"code": "deposed-object", "message": "Deposed object cleanup.", "paths": []}]} if {
	input.resource.deposed != null
}
