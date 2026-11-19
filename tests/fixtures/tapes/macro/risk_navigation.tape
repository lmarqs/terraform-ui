wait ready
resize 90 24
key R
wait ready
assert view [1/16]
assert view ^w wrap(off)
key G
wait ready
assert view [16/16]
assert view aws_instance.node_15
key g
wait ready
assert view [1/16]
key ctrl+w
wait ready
assert view ^w wrap(on)
key ctrl+w
wait ready
assert view ^w wrap(off)
key right
wait ready
assert view [1/16]
key left
wait ready
assert view aws_instance.node_00
key q
wait ready
