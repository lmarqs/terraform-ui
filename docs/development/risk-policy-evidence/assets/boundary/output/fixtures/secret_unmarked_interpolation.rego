package tfui.risk.v1
import rego.v1
classification := {"level": "low", "reasons": [{"code": "synthetic", "message": sprintf("%v", [input.token]), "paths": []}]}
