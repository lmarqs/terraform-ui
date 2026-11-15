package tfui.risk.v1
import rego.v1
classification := {"level": "low", "reasons": [{"code": "synthetic", "message": "synthetic", "paths": []}]}
unused if { print(input.token) }
