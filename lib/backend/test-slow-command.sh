#!/bin/sh

# This is a long running command with only intermittent output
# which can be used to test the system.

for i in 1 2 3 4
do
    echo "test-command.sh :: $i"
    sleep 1s
done

echo "test-command.sh :: All done"
