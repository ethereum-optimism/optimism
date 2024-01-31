set -e

for i in {1..100}; do
  echo "======================="
  echo "Running iteration $i"
  gotestsum -- -run 'TestControlLoop' ./... --count=1 --timeout=5s -race
  if [ $? -ne 0 ]; then
    echo "Test failed"
    exit 1
  fi
done
