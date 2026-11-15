wait ready
resize 90 24
key R
wait ready
assert view [1/16]
assert view Risk: medium — explicit policy classifications.
assert view -/+ aws_instance.node_00
assert view +4 more
key enter
wait ready
assert view delete-then-create
assert view nested: (sensitive)
resize 110 40
wait ready
assert view computed:
assert view (known after apply)
key G
wait ready
assert view (known after apply)
key esc
wait ready
key down
wait ready
assert view [2/16]
assert view +/- aws_instance.node_01
key enter
wait ready
assert view create-then-delete
key esc
wait ready
key down
wait ready
key enter
wait ready
assert view aws_instance.node_03
assert view - delete
key esc
wait ready
key down
wait ready
key enter
wait ready
assert view aws_instance.node_02
assert view + create
key esc
wait ready
key down
wait ready
key enter
wait ready
assert view aws_instance.node_04
assert view ~ update
key esc
wait ready
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
wait ready
assert view [16/16]
assert view aws_instance.node_15
resize 50 18
wait ready
assert view aws_instance.node_15
resize 110 30
wait ready
assert view aws_instance.node_15
key enter
wait ready
assert view Attributes:
key esc
wait ready
key q
