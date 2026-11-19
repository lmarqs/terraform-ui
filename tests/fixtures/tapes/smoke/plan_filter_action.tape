wait ready
key p
wait view aws_instance.web
key /
key ~
assert view aws_s3_bucket.data
assert view (1/2)
key backspace
key +
assert view aws_instance.web
assert view (1/2)
