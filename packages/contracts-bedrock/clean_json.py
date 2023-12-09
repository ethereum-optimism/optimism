import sys
import json

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
