# Pin and targeted apply: pin specific resources, apply only those
resize 120 35
wait ready
sleep 300ms

# Open plan
key p
wait ready
sleep 1s

# Pin first resource
key space
sleep 400ms

# Move down and pin another
key down
sleep 200ms
key down
sleep 200ms
key space
sleep 400ms

# Move down more, pin a third
key down
sleep 200ms
key down
sleep 200ms
key space
sleep 800ms

# Trigger targeted apply (pinned only)
key a
sleep 1.5s
