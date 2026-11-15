resize 140 40
wait ready
key p
wait ready
assert view aws_instance.web
assert view Overall risk: HIGH
key :
key b
key l
key a
key s
key t
key r
key a
key d
key i
key u
key s
key enter
wait ready
assert view High blast radius
assert view 2 total change(s)
key enter
assert view aws_s3_bucket.data
assert view [HIGH]
assert view Explicit built-in action/type classification for this mutation.
assert view Resource creation; the built-in level follows the resource type.
key q
