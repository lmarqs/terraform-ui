# State browser: list, filter, inspect
resize 120 35
wait ready
sleep 300ms

# Open state browser
key s
wait ready
sleep 1s

# Browse down
key down
sleep 200ms
key down
sleep 200ms
key down
sleep 200ms
key down
sleep 500ms

# Inspect a resource
key enter
sleep 2s

# Back to list
key esc
sleep 500ms

# Activate filter
key /
sleep 300ms

# Type filter query
key d
key b
sleep 800ms
