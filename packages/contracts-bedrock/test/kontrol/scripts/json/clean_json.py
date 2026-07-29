"""
Description:
    Unescapes the JSON produced by the stateDiff modifier
    defined in contracts-bedrock/scripts/deploy/Deploy.s.sol
    This script is used in ../make-summary-deployment.sh

Usage:
    After producing a state diff JSON with the stateDiff modifier
    (e.g. by executing KontrolDeployment::runKontrolDeployment), the JSON is
    saved under contracts-bedrock/snapshots/state-diff/Deploy.json. To unescape
    it, run: python3 clean_json.py $path_to_state_diff_json
    The unescaped JSON will rewrite the input file
"""

import sys

def clean_json(input_file):
    with open(input_file, 'r') as file:
        input_string = file.read()

    result = input_string

    # Remove backslashes
    result = result.replace('\\', '')

    # Remove " between ] and ,
    result = result.replace(']",', '],')

    # Remove " between } and ]
    result = result.replace('}"]', '}]')

    # Remove " between [ and {
    result = result.replace('["{', '[{')

    # Remove " between } and ,
    result = result.replace('}",', '},')

    # Remove " between , and {
    result = result.replace(',"{', ',{')

    # Remove " before [{
    result = result.replace('"[{', '[{')

    # Remove " after }]
    result = result.replace('}]"', '}]')

    with open(input_file, 'w') as file:
        file.write(result)

if __name__ == "__main__":
    # Check if a file path is provided as a command line argument
    if len(sys.argv) != 2:
        print("Usage: clean_json.py <file_path>")
        sys.exit(1)

    input_file_path = sys.argv[1]

    clean_json(input_file_path)

    print(f"Operation completed. Result saved to {input_file_path}")
