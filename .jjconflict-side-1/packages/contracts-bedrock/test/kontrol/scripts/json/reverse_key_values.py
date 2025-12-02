"""
Description:
    Reverses the key-value pairs of a given JSON
    The use case for this script within the project is to reverse the key-value
    pairs of the auto generated file contracts-bedrock/deployments/hardhat/.deployment
    so that it can be fed as the `--contract-names` argument to `kontrol summary`
    This script is used in ../make-summary-deployment.sh

Usage:
    To reverse the order of an $input_file and save it to an $output_file run
    `python3 reverse_key_values.py $input_file $output_file`
"""

import sys
import json

def reverse_json(input_file, output_file):
    try:
        with open(input_file, 'r') as file:
            json_data = json.load(file)

        reversed_json = {str(value): key[0].lower() + key[1:] for key, value in json_data.items()}

        with open(output_file, 'w') as file:
            json.dump(reversed_json, file, indent=2)

        print(f"Reversed JSON saved to {output_file}")

    except FileNotFoundError:
        print(f"Error: File not found: {input_file}")
        sys.exit(1)
    except json.JSONDecodeError:
        print(f"Error: Invalid JSON format in file: {input_file}")
        sys.exit(1)

if __name__ == "__main__":
    # Check if both input and output file paths are provided
    if len(sys.argv) != 3:
        print("Usage: reverse_key_values.py <input_file_path> <output_file_path>")
        sys.exit(1)

    input_file_path = sys.argv[1]
    output_file_path = sys.argv[2]

    # Execute the function to reverse JSON and save to the output file
    reverse_json(input_file_path, output_file_path)
