# Import: import an existing resource into state
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

# Trigger import
key n
sleep 500ms

# Type resource ID
key i
key -
key 1
key 2
key 3
key 4
key 5
key 6
sleep 500ms

# Confirm
key enter
sleep 2s
