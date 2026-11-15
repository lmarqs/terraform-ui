package tfui.risk.v1

import rego.v1

default classification := {
    "level": "low",
    "reasons": [{
        "code": "project-default",
        "message": "Explicit project default for this mutation; review this default before adoption.",
        "paths": [],
    }],
}

lambda_function if { input.resource.type == "aws_lambda_function" }
lambda_layer if { input.resource.type == "aws_lambda_layer_version" }
destructive if { input.change.actions[_] == "delete" }
lambda_update if { lambda_function; input.change.actions == ["update"] }

function_level := "critical" if { destructive
} else := "medium" if { input.change.actions == ["create"]
} else := "high" if { input.change.actions == ["update"] }

layer_level := "medium" if { destructive
} else := "low" if { input.change.actions == ["create"]
} else := "low" if { input.change.actions == ["update"] }

classification := {"level": function_level, "reasons": reasons} if { lambda_function }
classification := {"level": layer_level, "reasons": reasons} if { lambda_layer }

code_fields := ["filename", "s3_bucket", "s3_key", "s3_object_version", "image_uri", "source_code_hash"]
sides := ["before", "after"]
known_kinds := {"absent", "null", "string", "number", "boolean"}
opaque_kinds := {"unknown", "sensitive", "sensitive_unknown"}

changed_code_fields := [field |
    field := code_fields[_]
    before := object.get(input.change.before.entries, field, {"kind": "absent"})
    after := object.get(input.change.after.entries, field, {"kind": "absent"})
    known_kinds[before.kind]
    known_kinds[after.kind]
    before != after
]

code_paths := [[side, field] |
    field := changed_code_fields[_]
    side := sides[_]
    input.change[side].entries[field]
]

source_uncertain if {
    side := sides[_]
    opaque_kinds[input.change[side].kind]
}
source_uncertain if {
    side := sides[_]
    field := code_fields[_]
    node := input.change[side].entries[field]
    opaque_kinds[node.kind]
}

configuration_fields contains field if {
    side := sides[_]
    input.change[side].entries[field]
    before := object.get(input.change.before.entries, field, {"kind": "absent"})
    after := object.get(input.change.after.entries, field, {"kind": "absent"})
    before != after
}
configuration_paths := [[side, field] |
    configuration_fields[field]
    side := sides[_]
    input.change[side].entries[field]
]
compact_configuration_paths := [path |
    path := configuration_paths[i]
    i < 16
]

reasons := [{
    "code": "mutation-action",
    "message": "Destructive mutation; the example retains the reviewed action level and original action order.",
    "paths": [],
}] if { destructive
} else := [{
    "code": "mutation-action",
    "message": "Resource creation; the example retains the reviewed action level.",
    "paths": [],
}] if { input.change.actions == ["create"]
} else := [{
    "code": "lambda-code-source-change",
    "message": concat("", ["Known code-source attributes changed: ", concat(", ", changed_code_fields), ". This describes attribute evidence, not deployed bytes or deployment ownership."]),
    "paths": code_paths,
}] if { lambda_update; count(changed_code_fields) > 0
} else := [{
    "code": "lambda-source-uncertain",
    "message": "Code-source evidence is unknown or sensitive. Observed configuration changes do not establish code equality; function updates retain the reviewed high classification.",
    "paths": compact_configuration_paths,
}] if { lambda_update; source_uncertain
} else := [{
    "code": "lambda-configuration-update",
    "message": "Configuration attribute evidence changed; function updates retain the reviewed high classification. This does not establish a code upload.",
    "paths": compact_configuration_paths,
}] if { lambda_update; count(configuration_paths) > 0
} else := [{
    "code": "type-action-default",
    "message": "Explicit action/type classification for this Lambda mutation.",
    "paths": [],
}]

