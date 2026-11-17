package tfui.risk.v1

import rego.v1

classification := {"level": "low", "reasons": [{"code": "import-id-hidden", "message": "The import ID reaches the policy as a sensitive node.", "paths": []}]} if {
	input.change.importing == {"id": {"kind": "sensitive"}, "unknown": false}
}

classification := {"level": "high", "reasons": [{"code": "import-id-exposed", "message": "The import ID reached the policy in another form.", "paths": []}]} if {
	input.change.importing != {"id": {"kind": "sensitive"}, "unknown": false}
}
