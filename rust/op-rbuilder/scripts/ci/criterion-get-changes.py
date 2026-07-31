#!/usr/bin/env python3
#
# This script extracts the specific changes from a Criterion-generated comparison report.
#

# Require one argument
import sys
import os
import json
import argparse

from urllib.parse import quote

parser = argparse.ArgumentParser(description="Extract Criterion.rs benchmark changes.")
parser.add_argument(
    "report_dir", type=str, help="The directory containing the Criterion.rs report."
)

# add a --output-format argument to the parser
parser.add_argument(
    "--output-format",
    type=str,
    help="The output format.",
    default="txt",
    choices=["md", "txt", "html"],
)

# add a --only-significant argument to the parser
parser.add_argument(
    "--only-significant",
    action="store_true",
    help="Only show significant changes.",
)

args = parser.parse_args()
# print(args)

report_dir = args.report_dir
output_format = args.output_format
only_significant = args.only_significant

# Make sure the directory exists
if not os.path.isdir(report_dir):
    print(f"Error: directory '{report_dir}' does not exist.")
    sys.exit(1)


# Find and parse all benchmark changes (based on "change/estimates.json" files)
changes = []
for root, dirs, files in os.walk(report_dir):
    for file in files:
        if root.endswith("/change") and file == "estimates.json":
            change_estimates = json.load(open(os.path.join(root, file), "r"))
            benchmark_info = json.load(
                open(os.path.join(root, "../new/benchmark.json"), "r")
            )
            changes.append(
                {
                    "dir": root,
                    "file": file,
                    "estimates": change_estimates,
                    "benchmark": benchmark_info,
                }
            )

# Changes are considered non-noise if above a 2% default threshold. See also
# the Criterion docs: https://bheisler.github.io/criterion.rs/book/user_guide/command_line_output.html#change
# and the Criterion code: https://github.com/bheisler/criterion.rs/blob/master/src/report.rs#L596
mean_p_noise_threshold = 0.02


def has_improved(estimates_data):
    """
    Compare the mean confidence to the noise threshold. See also Criterion reference code:
    https://github.com/bheisler/criterion.rs/blob/master/src/report.rs#L779
    """
    confidence_interval = estimates_data["mean"]["confidence_interval"]
    lower_bound = confidence_interval["lower_bound"]
    upper_bound = confidence_interval["upper_bound"]
    if lower_bound < -mean_p_noise_threshold and upper_bound < -mean_p_noise_threshold:
        return True
    elif lower_bound > mean_p_noise_threshold and upper_bound > mean_p_noise_threshold:
        return False
    return None


def estimate_to_str(estimates_data):
    """
    Summarize whether an improvement or degradation has occurred.
    Returns: (message, is_significant)
    """
    mean_p = estimates_data["mean"]["point_estimate"]
    if abs(mean_p) < mean_p_noise_threshold:
        return "No change in performance detected.", False
    has_improved_str = has_improved(estimates_data)
    if has_improved_str is None:
        return "Change within noise threshold.", False
    elif has_improved_str:
        return f"Performance has improved.", True
    else:
        return f"Performance has degraded.", True


# Prepare changes for printing
change_data = []
for change in changes:
    full_id = change["benchmark"]["full_id"]
    url = f"{os.getenv('PUBLIC_URL_REPORT')}/{quote(full_id)}/report/index.html"
    mean_p = change["estimates"]["mean"]["point_estimate"]
    mean_p_fmt = f"{100 * mean_p:.2f}"
    msg, is_significant = estimate_to_str(change["estimates"])
    if only_significant and not is_significant:
        continue
    change_data.append((full_id, url, mean_p_fmt, msg, is_significant))

if not change_data:
    exit(0)

# Print the changes
if output_format == "md":
    print("| Benchmark | Mean | Status |")
    print("| --- | --- | --- |")
    for full_id, url, mean_fmt, msg, is_significant in change_data:
        if is_significant:
            msg = f"**{msg}**"
        print(f"| [{full_id}]({url}) | {mean_fmt}% | {msg} |")
elif output_format == "html":
    print(
        "<thead><tr><th>Benchmark</th><th>Mean</th><th>Status</th></tr></thead><tbody>"
    )
    for full_id, url, mean_fmt, msg, is_significant in change_data:
        if is_significant:
            msg = f"<strong>{msg}</strong>"
        print(
            f"<tr><td><a href='{url}'>{full_id}</a></td><td>{mean_fmt}%</td><td>{msg}</td></tr>"
        )
    print("</tbody>")
else:
    print("Benchmark \t Mean \t Status \t Significant")
    for full_id, url, mean_fmt, msg, is_significant in change_data:
        print(f"{full_id} \t {mean_fmt}% \t {msg} \t {is_significant}")
