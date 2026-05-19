# Plan review: tree navigation, expand details, risk badges visible
resize 80 20
wait ready
sleep 300ms

# Open plan view
key p
wait ready
sleep 1s

# Scroll down to see all changes with risk badges
key down
sleep 200ms
key down
sleep 200ms
key down
sleep 200ms
key down
sleep 200ms
key down
sleep 200ms
key down
sleep 200ms
key down
sleep 200ms
key down
sleep 200ms
key down
sleep 500ms

# Go to the CRITICAL RDS delete and inspect it
key enter
sleep 2s

# Back to list
key esc
sleep 500ms

# Switch to tree view
key ctrl+t
sleep 1s
