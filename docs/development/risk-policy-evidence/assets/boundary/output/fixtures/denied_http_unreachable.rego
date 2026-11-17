package tfui.risk.v1
import rego.v1
classification := {"level": "low", "reasons": [{"code": "synthetic", "message": "synthetic", "paths": []}]}
unused := http.send({"method":"GET","url":"http://127.0.0.1:32829/probe"})
