# Force unlock: remove a stuck state lock
resize 120 35
wait ready
sleep 300ms

# Open plan to trigger lock detection
key p
wait ready
sleep 1s

# Navigate to force-unlock via command mode
key :
sleep 300ms
key f
key o
key r
key c
key e
key u
key n
key l
key o
key c
key k
sleep 300ms
key enter
sleep 1s

# Confirm unlock
key y
sleep 2s
