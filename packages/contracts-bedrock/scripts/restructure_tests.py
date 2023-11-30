import os
import shutil


def mimic_directory_structure(src_folder: str, test_folder: str) -> None:
    """
    This function takes a source folder and a test folder as input, and restructures
    the test folder to match the directory structure of the source folder.

    Only moves test files ("<name>.t.sol") at the root level of the `test` folder.
    """

    # Walk through the src folder and collect a list of all .sol files
    sol_files = []
    for root, _, files in os.walk(src_folder):
        for file in files:
            if file.endswith(".sol"):
                sol_files.append(os.path.join(root, file))

    # Iterate through each .t.sol file in the test folder
    for test_file in os.listdir(test_folder):
        if test_file.endswith(".t.sol"):
            # Construct the corresponding .sol file name
            sol_file = test_file.replace(".t.sol", ".sol")

            # Find the full path of the corresponding .sol file in the src folder
            src_path = None
            for sol_path in sol_files:
                if sol_path.endswith(os.path.sep + sol_file):
                    src_path = sol_path
                    break

            if src_path:
                # Calculate the relative path from the src folder to the .sol file
                rel_path = os.path.relpath(src_path, src_folder)

                # Construct the destination path within the test folder
                dest_path = os.path.join(
                    test_folder, rel_path).replace(".sol", ".t.sol")

                # Create the directory structure if it doesn't exist
                dest_dir = os.path.dirname(dest_path)
                os.makedirs(dest_dir, exist_ok=True)

                # Copy the .t.sol file to the destination folder
                shutil.move(os.path.join(test_folder, test_file), dest_path)
                print(f"Moved {test_file} to {dest_path}")
            else:
                print(f"No corresponding .sol file found for {test_file}")


# Specify the source and test folder paths
src_folder = "src"
test_folder = "test"

# Call the mimic_directory_structure function
mimic_directory_structure(src_folder, test_folder)
