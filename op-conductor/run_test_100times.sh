set -e

for i in {1..100}; do
  echo "======================="
  echo "Running iteration $i"
  # gotestsum ./... --count=1
  gotestsum -- -run 'TestControlLoop' ./... --count=1 --timeout=5s -race
  if [ $? -ne 0 ]; then
    echo "Command failed"
  fi
done
