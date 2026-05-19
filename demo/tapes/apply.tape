# Apply: pin resource and trigger targeted apply
resize 80 20
wait ready
sleep 300ms

# Open plan view
key p
wait ready
sleep 1s

# Pin a resource for targeted apply
key down
sleep 200ms
key down
sleep 200ms
key space
sleep 800ms

# Trigger apply
key a
sleep 1.5s

# Confirm
key y
sleep 2s
