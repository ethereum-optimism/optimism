#!/usr/bin/env python3
#
# Customize the Criterion.rs report HTML files (adding style, links and other information)
#
# Usage:
#   $ python3 criterion-update-html.py <input-file> [<output-file>]
#
# Criterion HTML templates: https://github.com/bheisler/criterion.rs/tree/master/src/html
#

import sys
import re
import os
from datetime import datetime, timezone

if len(sys.argv) < 2:
    print(f"Usage: {sys.argv[0]} <input-file> [<output-file>]")
    sys.exit(1)

# Prepare variables
input_file = sys.argv[1]
output_file = sys.argv[2] if len(sys.argv) > 2 else None
script_dir = os.path.dirname(os.path.realpath(__file__))

# Prepare paths
fn_styles = f"{script_dir}/templates/report-styles.css"
fn_template_head = f"{script_dir}/templates/report-head.html"
fn_template_footer = f"{script_dir}/templates/report-footer.html"
fn_template_index = f"{script_dir}/templates/report-index.html"
fn_template_report = f"{script_dir}/templates/report-criterion-benchmark.html"
fn_partial_index_changes = f"{script_dir}/templates/partials/index-changes.html"

# Prepare templates
template_head = open(fn_template_head, "r").read()
template_footer = open(fn_template_footer, "r").read()
template_index = open(fn_template_index, "r").read()
template_report = open(fn_template_report, "r").read()
partial_index_changes = open(fn_partial_index_changes, "r").read()

# Setup template variables
PR_NUMBER = os.getenv("PR_NUMBER")
PUBLIC_URL_REPORT = os.getenv("PUBLIC_URL_REPORT")
CHANGES_TABLE = os.getenv("BENCH_CHANGES_HTML")

# Template vars need to be tuple to ensure ordering
template_vars = [
    ("__TITLE__", "rbuilder benchmarks for __HEAD_SHA_SHORT__"),
    ("__STYLE__", "<style>" + open(fn_styles, "r").read() + "</style>"),
    ("__DATE__", os.getenv("DATE") or datetime.now(timezone.utc)),
    ("__HEAD_SHA__", os.getenv("HEAD_SHA")),
    ("__HEAD_SHA_SHORT__", os.getenv("HEAD_SHA_SHORT")),
    ("__HEAD_BRANCH__", os.getenv("HEAD_BRANCH")),
    ("__BASE_SHA__", os.getenv("BASE_SHA")),
    ("__BASE_REF__", os.getenv("BASE_REF")),
    ("__GITHUB_SHA__", os.getenv("GITHUB_SHA")),
    ("__GITHUB_REF_NAME__", os.getenv("GITHUB_REF_NAME")),
    ("__GITHUB_REF_TYPE__", os.getenv("GITHUB_REF_TYPE")),
    ("__GITHUB_HEAD_REF__", os.getenv("GITHUB_HEAD_REF")),
    ("__GITHUB_ACTOR__", os.getenv("GITHUB_ACTOR")),
    ("__RUNNER_ARCH__", os.getenv("RUNNER_ARCH")),
    ("__RUNNER_OS__", os.getenv("RUNNER_OS")),
    ("__PR_NUMBER__", PR_NUMBER),
    ("__REPO_URL__", os.getenv("REPO_URL")),
    ("__STATIC_URL__", os.getenv("PUBLIC_URL_STATIC")),
]


def remove_from_str(s, pattern):
    return re.sub(pattern, "", s, flags=re.DOTALL)


def render_template(template_html_body, content, template_vars_extra=[]):
    html = template_head + template_html_body

    core_content = [
        ("__FOOTER__", template_footer),
        ("__CONTENT__", content),
    ]

    # replace template variables
    for key, value in core_content + template_vars + template_vars_extra:
        html = re.sub(key, str(value), html)

    # some removals (i.e. if no PR number)
    if not PR_NUMBER:
        html = remove_from_str(html, r"""<tr id="head-info-PR">.*?</tr>""")

    return html


def update_index(data):
    url = f"{PUBLIC_URL_REPORT}/report/index.html"
    template_vars = [("__URL__", url)]

    # extract original content
    ul_element = re.search(r"<ul>(.*)</ul>", data, re.DOTALL).group(1)
    content = "<ul>" + ul_element + "</ul>"

    changes = ""
    if CHANGES_TABLE:
        changes = re.sub(r"__CHANGES_TABLE__", CHANGES_TABLE, partial_index_changes)
    template_vars.append(("__CHANGES__", changes))

    # Render template
    updated_html = render_template(template_index, content, template_vars)
    return updated_html


def update_subpage(data):
    # extract the body
    content = re.search(r"<body>(.*)</body>", data, re.DOTALL).group(1)

    # get the benchmark name
    benchmark_name = re.search(r"<h2>(.*?)</h2>", data, re.DOTALL).group(1)

    # remove <div id="footer">
    content = remove_from_str(content, r"""<div id="footer">.*</div>""")

    # Render template
    url = f"{PUBLIC_URL_REPORT}/{benchmark_name}/report/index.html"
    template_vars = [
        ("__URL__", url),
        (
            "__TITLE__",
            "rbuilder benchmark '__BENCHMARK_NAME__' for __HEAD_SHA_SHORT__",
        ),
        ("__BENCHMARK_NAME__", benchmark_name),
    ]
    updated_html = render_template(template_report, content, template_vars)
    return updated_html


f = open(input_file, "r")
data = f.read()
f.close()

# check if file contains "<h2>Criterion.rs Benchmark Index</h2>"
if re.search(r"<h2>Criterion.rs Benchmark Index</h2>", data):
    output = update_index(data)
else:
    output = update_subpage(data)

# write to output file
if output_file:
    f = open(output_file, "w")
    f.write(output)
    f.close()
else:
    print(output)
