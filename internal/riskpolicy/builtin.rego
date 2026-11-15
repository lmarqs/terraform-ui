package tfui.risk.v1

import rego.v1

critical_types := {
    "aws_db_instance",
    "aws_rds_cluster",
    "aws_rds_cluster_instance",
    "aws_dynamodb_table",
    "aws_s3_bucket",
    "aws_efs_file_system",
    "aws_fsx_lustre_file_system",
    "aws_fsx_windows_file_system",
    "aws_redshift_cluster",
    "aws_elasticache_cluster",
    "aws_elasticache_replication_group",
    "aws_docdb_cluster",
    "aws_neptune_cluster",
    "aws_kms_key",
    "aws_kms_alias",
    "google_sql_database_instance",
    "google_storage_bucket",
    "google_kms_key_ring",
    "google_kms_crypto_key",
    "azurerm_sql_server",
    "azurerm_cosmosdb_account",
    "azurerm_storage_account",
    "azurerm_key_vault",
}

high_types := {
    "aws_iam_role",
    "aws_iam_policy",
    "aws_iam_user",
    "aws_iam_group",
    "aws_iam_instance_profile",
    "aws_vpc",
    "aws_subnet",
    "aws_route_table",
    "aws_nat_gateway",
    "aws_internet_gateway",
    "aws_eip",
    "aws_lb",
    "aws_alb",
    "aws_elb",
    "aws_ecs_cluster",
    "aws_eks_cluster",
    "aws_lambda_function",
    "aws_cloudfront_distribution",
    "google_compute_network",
    "google_compute_subnetwork",
    "google_container_cluster",
    "google_project_iam_member",
    "azurerm_virtual_network",
    "azurerm_kubernetes_cluster",
    "azurerm_role_assignment",
}

medium_types := {
    "aws_security_group",
    "aws_security_group_rule",
    "aws_network_acl",
    "aws_route53_record",
    "aws_cloudwatch_log_group",
    "aws_sns_topic",
    "aws_sqs_queue",
    "aws_ecr_repository",
    "google_compute_firewall",
    "google_dns_record_set",
    "azurerm_network_security_group",
    "azurerm_dns_zone",
}

elevated if { critical_types[input.resource.type] }
elevated if { high_types[input.resource.type] }
destructive if { input.change.actions[_] == "delete" }
lambda_update if {
    input.resource.type == "aws_lambda_function"
    input.change.actions == ["update"]
}

level := "critical" if { destructive; elevated
} else := "high" if { destructive; medium_types[input.resource.type]
} else := "medium" if { destructive
} else := "high" if { input.change.actions == ["update"]; elevated
} else := "medium" if { input.change.actions == ["update"]; medium_types[input.resource.type]
} else := "low" if { input.change.actions == ["update"]
} else := "medium" if { input.change.actions == ["create"]; elevated
} else := "low" if { input.change.actions == ["create"] }

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
    "message": "Destructive mutation; the built-in level follows the resource type and original action order.",
    "paths": [],
}] if { destructive
} else := [{
    "code": "mutation-action",
    "message": "Resource creation; the built-in level follows the resource type.",
    "paths": [],
}] if { input.change.actions == ["create"]
} else := [{
    "code": "lambda-code-source-change",
    "message": concat("", ["Known code-source attributes changed: ", concat(", ", changed_code_fields), ". This describes attribute evidence, not deployed bytes or deployment ownership."]),
    "paths": code_paths,
}] if { lambda_update; count(changed_code_fields) > 0
} else := [{
    "code": "lambda-source-uncertain",
    "message": "Code-source evidence is unknown or sensitive. Observed configuration changes do not establish code equality; function updates retain the built-in high classification.",
    "paths": compact_configuration_paths,
}] if { lambda_update; source_uncertain
} else := [{
    "code": "lambda-configuration-update",
    "message": "Configuration attribute evidence changed; function updates retain the built-in high classification. This does not establish a code upload.",
    "paths": compact_configuration_paths,
}] if { lambda_update; count(configuration_paths) > 0
} else := [{
    "code": "type-action-default",
    "message": "Explicit built-in action/type classification for this mutation.",
    "paths": [],
}]

classification := {"level": level, "reasons": reasons}
