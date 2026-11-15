wait ready
resize 110 30
key R
wait ready
assert view Risk unavailable: 1 of 3 eligible mutations failed; 2 classified.
assert view risk unavailable: result_undefined
assert view [1/3]
assert view aws_instance.z_failed
key enter
wait ready
assert view Risk:
assert view unavailable: result_undefined
assert view result_undefined
key esc
wait ready
key down
wait ready
assert view [2/3]
key enter
wait ready
assert view aws_instance.a_create
assert view + create
assert view Successful neighboring mutation.
key esc
wait ready
key down
wait ready
key enter
wait ready
assert view aws_instance.b_delete
assert view - delete
key esc
wait ready
key q
