package tfui.risk.v1
import rego.v1
classification := {"level": "low", "reasons": [{"code": "classified-neighbor", "message": "Successful neighboring mutation.", "paths": []}]} if { input.resource.address != "aws_instance.z_failed" }
