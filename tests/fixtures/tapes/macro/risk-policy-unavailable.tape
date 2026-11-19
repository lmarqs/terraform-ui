wait ready
key p
wait ready
assert view aws_instance.web
assert view risk unavailable: source_unreadable
key enter
wait ready
assert view risk unavailable: source_unreadable
key esc
key q
key R
wait ready
assert view risk unavailable: source_unreadable
resize 45 16
wait ready
assert view risk unavailable: source_unreadable
