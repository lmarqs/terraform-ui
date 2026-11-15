resize 140 40
wait ready
key p
wait ready
assert view aws_instance.web
assert view risk unavailable: input_incomplete
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
assert view Blast radius unavailable: incomplete risk assessment
assert view 2 total change(s)
key enter
assert view risk unavailable: input_incomplete
key q
