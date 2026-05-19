# Hero demo: full workflow (home → plan → pin → targeted apply)
resize 80 20
wait ready
sleep 500ms

# Navigate to plan
key p
wait ready
sleep 1s

# Scroll through the changes
key down
sleep 300ms
key down
sleep 300ms
key down
sleep 300ms

# Inspect a resource (expand details)
key enter
sleep 1.5s

# Back to list
key esc
sleep 300ms

# Pin the critical RDS destroy for targeted apply
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
key space
sleep 800ms

# Pin the new DB too
key down
sleep 200ms
key space
sleep 800ms

# Trigger apply
key a
sleep 1.5s
