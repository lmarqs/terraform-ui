package tfui.risk.v1_test

import rego.v1
import data.tfui.risk.v1 as risk

test_filename_same_hash if {
    result := risk.classification with input as data.fixtures["filename-same-hash"]
    result.level == "high"
    count(result.reasons) == 1
    result.reasons[0].code == "lambda-code-source-change"
    result.reasons[0].paths == [["before", "filename"], ["after", "filename"]]
}

test_filename_null_hash if {
    result := risk.classification with input as data.fixtures["filename-null-hash"]
    result.level == "high"
    count(result.reasons) == 1
    result.reasons[0].code == "lambda-code-source-change"
    result.reasons[0].paths == [["before", "filename"], ["after", "filename"]]
}

test_filename_absent_hash if {
    result := risk.classification with input as data.fixtures["filename-absent-hash"]
    result.level == "high"
    count(result.reasons) == 1
    result.reasons[0].code == "lambda-code-source-change"
    result.reasons[0].paths == [["before", "filename"], ["after", "filename"]]
}

test_filename_unknown_hash if {
    result := risk.classification with input as data.fixtures["filename-unknown-hash"]
    result.level == "high"
    count(result.reasons) == 1
    result.reasons[0].code == "lambda-code-source-change"
    result.reasons[0].paths == [["before", "filename"], ["after", "filename"]]
}

test_hash_unknown if {
    result := risk.classification with input as data.fixtures["hash-unknown"]
    result.level == "high"
    count(result.reasons) == 1
    result.reasons[0].code == "lambda-source-uncertain"
}

test_memory_known_sources if {
    result := risk.classification with input as data.fixtures["memory-known-sources"]
    result.level == "high"
    count(result.reasons) == 1
    result.reasons[0].code == "lambda-configuration-update"
    result.reasons[0].paths == [["before", "memory_size"], ["after", "memory_size"]]
}

test_memory_unknown_source if {
    result := risk.classification with input as data.fixtures["memory-unknown-source"]
    result.level == "high"
    count(result.reasons) == 1
    result.reasons[0].code == "lambda-source-uncertain"
}

test_sensitive_source if {
    result := risk.classification with input as data.fixtures["sensitive-source"]
    result.level == "high"
    count(result.reasons) == 1
    result.reasons[0].code == "lambda-source-uncertain"
}

test_function_create if {
    result := risk.classification with input as data.fixtures["function-create"]
    result.level == "medium"
    count(result.reasons) == 1
    result.reasons[0].code == "mutation-action"
    result.reasons[0].paths == []
}

test_function_delete if {
    result := risk.classification with input as data.fixtures["function-delete"]
    result.level == "critical"
    count(result.reasons) == 1
    result.reasons[0].code == "mutation-action"
    result.reasons[0].paths == []
}

test_function_delete_create if {
    result := risk.classification with input as data.fixtures["function-delete-create"]
    result.level == "critical"
    count(result.reasons) == 1
    result.reasons[0].code == "mutation-action"
    result.reasons[0].paths == []
}

test_function_create_delete if {
    result := risk.classification with input as data.fixtures["function-create-delete"]
    result.level == "critical"
    count(result.reasons) == 1
    result.reasons[0].code == "mutation-action"
    result.reasons[0].paths == []
}

test_layer_create if {
    result := risk.classification with input as data.fixtures["layer-create"]
    result.level == "low"
    count(result.reasons) == 1
    result.reasons[0].code == "mutation-action"
    result.reasons[0].paths == []
}

test_layer_delete if {
    result := risk.classification with input as data.fixtures["layer-delete"]
    result.level == "medium"
    count(result.reasons) == 1
    result.reasons[0].code == "mutation-action"
    result.reasons[0].paths == []
}

test_layer_delete_create if {
    result := risk.classification with input as data.fixtures["layer-delete-create"]
    result.level == "medium"
    count(result.reasons) == 1
    result.reasons[0].code == "mutation-action"
    result.reasons[0].paths == []
}

test_layer_create_delete if {
    result := risk.classification with input as data.fixtures["layer-create-delete"]
    result.level == "medium"
    count(result.reasons) == 1
    result.reasons[0].code == "mutation-action"
    result.reasons[0].paths == []
}

test_layer_update if {
    result := risk.classification with input as data.fixtures["layer-update"]
    result.level == "low"
    count(result.reasons) == 1
    result.reasons[0].code == "type-action-default"
    result.reasons[0].paths == []
}

test_database_delete if {
    result := risk.classification with input as data.fixtures["database-delete"]
    result.level == "low"
    count(result.reasons) == 1
    result.reasons[0].code == "project-default"
    result.reasons[0].paths == []
}
