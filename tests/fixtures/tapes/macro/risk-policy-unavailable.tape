wait ready
key p
wait ready
assert view aws_instance.web
assert view risk unavailable: source_unreadable
key enter
wait ready
assert view risk unavailable: source_unreadable
key esc
key esc
key R
wait ready
assert view risk unavailable: source_unreadable
assert view ^r retry
resize 45 16
wait ready
assert view risk unavailable: source_unreadable
