wait ready
resize 100 30
key R
wait ready
assert view input: {"env":"prod","id":"a"} → {"env":"stage","id":(known after apply)}
assert view output: {"env":"prod","id":"a"} → (known after apply)
key esc
wait ready
key p
wait ready
key down
key enter
wait ready
assert view terraform_data.b
assert view + {"env":"stage","id":(known after apply)}
