# Taint: mark a resource for recreation
resize 80 20
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

# Trigger taint
key t
sleep 500ms

# Confirm
key y
sleep 2s
