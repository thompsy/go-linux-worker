#!/bin/sh

# This is a long running command with only intermittent output
# which can be used to test the system.

for i in 1 2 3 4 5 6 7 8 9 10
do
    echo "test-command.sh :: pid=$$ :: $i"
    sleep 2s
done

echo "test-command.sh :: All done"
