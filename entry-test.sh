#!/usr/bin/env sh


# We need to wait some time to let health-check do its job
echo "Waiting for 10 seconds while the servers are configured..."
sleep 10
# Run our tests and benchmarks
bood -v out/.integration-tests.test.out
# And print them out
cat out/.integration-tests.test.out
