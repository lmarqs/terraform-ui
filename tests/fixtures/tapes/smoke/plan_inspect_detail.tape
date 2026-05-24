wait ready
key p
wait view +
key down
key enter
sleep 100ms
screenshot /tmp/plan_inspect_detail.txt
assert view Action
assert view Address
key esc
assert view +
