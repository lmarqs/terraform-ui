wait ready
resize 90 24
key R
wait ready
assert view [1/16]
assert view -/+ aws_instance.node_00
assert view +2 more
key enter
assert view delete-then-create
assert view (sensitive)
key G
assert view (known after apply)
key esc
key down
assert view [2/16]
assert view +/- aws_instance.node_01
key enter
assert view create-then-delete
key esc
key down
key down
key down
key down
key down
key down
key down
key down
key down
key down
key down
key down
key down
key down
assert view [16/16]
assert view aws_instance.node_15
resize 50 18
assert view aws_instance.node_15
resize 110 30
assert view aws_instance.node_15
key enter
assert view Attributes:
key esc
key q
