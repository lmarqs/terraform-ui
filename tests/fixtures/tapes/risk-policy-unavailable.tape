wait ready
key p
assert view aws_instance.web
assert view risk unavailable: source_unreadable
key enter
assert view risk unavailable: source_unreadable
key esc
key q
key R
assert view risk unavailable: source_unreadable
resize 45 16
assert view risk unavailable: source_unreadable
