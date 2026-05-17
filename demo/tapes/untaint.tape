# Untaint: remove taint from a resource
resize 120 35
wait ready
sleep 300ms

# Open state browser
key s
wait ready
sleep 1s

# Navigate to a resource
key down
sleep 200ms
key down
sleep 200ms
key down
sleep 300ms

# Trigger untaint
key T
sleep 500ms

# Confirm
key y
sleep 2s
