#!/usr/bin/env python3
"""
Fetch CircleCI 'main' workflow runtime data from GitHub Checks API
and write main-workflow-runtime-data.json + main-workflow-runtime.svg.

Usage:
    python3 docs/ci-metrics/fetch-runtime.py [--days N]

Requirements: Python 3.8+, no external dependencies.
Note: unauthenticated GitHub API limit is 60 req/hr; set GITHUB_TOKEN env var
to raise it to 5000.
"""

import json
import os
import sys
import time
import urllib.request
from collections import Counter, defaultdict
from datetime import datetime, timedelta, timezone

REPO = "ethereum-optimism/optimism"
BRANCH = "develop"
WORKFLOW_NAME = "main"
CIRCLECI_APP = "CircleCI Checks"


def fetch(url, token=None, retry=3):
    headers = {"Accept": "application/vnd.github.v3+json"}
    if token:
        headers["Authorization"] = f"token {token}"
    for attempt in range(retry):
        try:
            req = urllib.request.Request(url, headers=headers)
            with urllib.request.urlopen(req) as r:
                remaining = r.headers.get("X-RateLimit-Remaining", "?")
                data = json.loads(r.read())
                return data, int(remaining) if remaining != "?" else 60
        except Exception as e:
            if attempt < retry - 1:
                time.sleep(2 ** (attempt + 1))
            else:
                raise


def collect_data(days=7, token=None):
    since = (datetime.now(timezone.utc) - timedelta(days=days)).isoformat().replace("+00:00", "Z")
    print(f"Fetching commits on {BRANCH} since {since[:10]}...")

    commits_url = (
        f"https://api.github.com/repos/{REPO}/commits"
        f"?sha={BRANCH}&per_page=50&since={since}"
    )
    commits_data, remaining = fetch(commits_url, token)
    print(f"Got {len(commits_data)} commits, {remaining} API calls remaining")

    results = []
    for i, commit in enumerate(commits_data):
        sha = commit["sha"]
        commit_date = commit["commit"]["committer"]["date"]

        runs_url = (
            f"https://api.github.com/repos/{REPO}/commits/{sha}"
            f"/check-runs?check_name={WORKFLOW_NAME}&per_page=10"
        )
        runs_data, remaining = fetch(runs_url, token)
        runs = runs_data.get("check_runs", [])

        main_run = None
        for r in runs:
            app = r.get("app") or {}
            if CIRCLECI_APP in app.get("name", "") and r["name"] == WORKFLOW_NAME:
                main_run = r
                break

        if main_run:
            started = main_run.get("started_at")
            completed = main_run.get("completed_at")
            conclusion = main_run.get("conclusion", "unknown")
            duration_min = None
            if started and completed:
                s = datetime.fromisoformat(started.replace("Z", "+00:00"))
                e = datetime.fromisoformat(completed.replace("Z", "+00:00"))
                duration_min = (e - s).total_seconds() / 60
            results.append({
                "sha": sha[:12],
                "commit_date": commit_date[:16],
                "started_at": started,
                "conclusion": conclusion,
                "duration_min": duration_min,
            })
            dur_str = f"{duration_min:.1f}m" if duration_min else "no timing"
            print(f"  [{i+1:2d}] {sha[:12]} {commit_date[:19]}  {conclusion:<10} {dur_str}  (API: {remaining} left)")
        else:
            print(f"  [{i+1:2d}] {sha[:12]} {commit_date[:19]}  (no main workflow)")

        time.sleep(0.3)

    return results


def make_svg(data, out_path):
    data = sorted(data, key=lambda r: r.get("started_at") or r["commit_date"])

    width, height = 960, 460
    ml, mr, mt, mb = 65, 55, 75, 90
    plot_w = width - ml - mr
    plot_h = height - mt - mb

    t_min = datetime.fromisoformat(data[0]["started_at"].replace("Z", "+00:00")).replace(
        hour=0, minute=0, second=0, microsecond=0
    )
    t_max = t_min + timedelta(days=8)
    t_range = (t_max - t_min).total_seconds()
    d_max = 40

    def tx(t):
        return ml + (t - t_min).total_seconds() / t_range * plot_w

    def ty(d):
        return mt + plot_h - d / d_max * plot_h

    times = [datetime.fromisoformat(r["started_at"].replace("Z", "+00:00")) for r in data]
    full_runs = [r for r in data if r.get("duration_min") and r["duration_min"] >= 5]
    avg = sum(r["duration_min"] for r in full_runs) / max(len(full_runs), 1)
    min_d = min(r["duration_min"] for r in full_runs)
    max_d = max(r["duration_min"] for r in full_runs)
    counts = Counter(r["conclusion"] for r in data)
    date_range = f"{times[0].strftime('%b %d')}–{times[-1].strftime('%b %d, %Y')}"

    colors = {"success": "#22c55e", "failure": "#f87171", "cancelled": "#fbbf24"}

    svg = []
    svg.append(
        f'<svg xmlns="http://www.w3.org/2000/svg" width="{width}" height="{height}" '
        f'viewBox="0 0 {width} {height}" style="background:#0f172a;font-family:\'Courier New\',monospace">'
    )
    svg.append(f'<rect width="{width}" height="{height}" fill="#0f172a"/>')
    svg.append(f'<rect x="{ml}" y="{mt}" width="{plot_w}" height="{plot_h}" fill="#1e293b" rx="4"/>')

    summary = f"{len(data)} runs: {counts.get('success',0)} success, {counts.get('failure',0)} failed, {counts.get('cancelled',0)} cancelled"
    svg.append(f'<text x="{width//2}" y="28" fill="#f1f5f9" font-size="15" text-anchor="middle" font-weight="bold">'
               f"CircleCI 'main' Workflow — Wall-Clock Runtime (develop branch)</text>")
    svg.append(f'<text x="{width//2}" y="50" fill="#94a3b8" font-size="11" text-anchor="middle">'
               f"{date_range}  ·  {summary}  ·  Avg {avg:.1f}m  ·  Range {min_d:.1f}m–{max_d:.1f}m</text>")

    for d in range(0, 45, 5):
        y = ty(d)
        svg.append(f'<line x1="{ml}" y1="{y:.1f}" x2="{ml+plot_w}" y2="{y:.1f}" '
                   f'stroke="#334155" stroke-width="{"1.5" if d % 10 == 0 else "0.7"}"/>')
        svg.append(f'<text x="{ml-8}" y="{y+4:.1f}" fill="#64748b" font-size="10" text-anchor="end">{d}m</text>')

    svg.append(f'<line x1="{ml}" y1="{mt}" x2="{ml}" y2="{mt+plot_h}" stroke="#475569" stroke-width="1.5"/>')
    svg.append(f'<line x1="{ml}" y1="{mt+plot_h}" x2="{ml+plot_w}" y2="{mt+plot_h}" stroke="#475569" stroke-width="1.5"/>')
    svg.append(f'<text x="14" y="{mt + plot_h//2}" fill="#94a3b8" font-size="11" text-anchor="middle" '
               f'transform="rotate(-90 14 {mt + plot_h//2})">Duration (minutes)</text>')

    cur = t_min
    day_idx = 0
    while cur <= t_max:
        x = tx(cur)
        if ml <= x <= ml + plot_w:
            svg.append(f'<line x1="{x:.1f}" y1="{mt}" x2="{x:.1f}" y2="{mt+plot_h}" stroke="#334155" stroke-width="1"/>')
            svg.append(f'<text x="{x:.1f}" y="{mt+plot_h+16}" fill="#64748b" font-size="10" text-anchor="middle">'
                       f'{cur.strftime("%b %d")}</text>')
        if day_idx % 2 == 1:
            x1 = max(ml, tx(cur))
            x2 = min(ml + plot_w, tx(cur + timedelta(days=1)))
            svg.append(f'<rect x="{x1:.1f}" y="{mt}" width="{max(0,x2-x1):.1f}" height="{plot_h}" fill="rgba(255,255,255,0.02)"/>')
        cur += timedelta(days=1)
        day_idx += 1

    y_avg = ty(avg)
    svg.append(f'<line x1="{ml}" y1="{y_avg:.1f}" x2="{ml+plot_w}" y2="{y_avg:.1f}" '
               f'stroke="#475569" stroke-width="1.5" stroke-dasharray="6,3"/>')
    svg.append(f'<text x="{ml+plot_w+4}" y="{y_avg+4:.1f}" fill="#64748b" font-size="10">avg {avg:.1f}m</text>')

    pts = [(tx(times[i]), ty(data[i]["duration_min"] or 0), data[i].get("duration_min") or 0)
           for i in range(len(data))]
    for i in range(len(pts) - 1):
        x1, y1, d1 = pts[i]
        x2, y2, d2 = pts[i + 1]
        if d1 >= 5 and d2 >= 5:
            svg.append(f'<line x1="{x1:.1f}" y1="{y1:.1f}" x2="{x2:.1f}" y2="{y2:.1f}" '
                       f'stroke="#475569" stroke-width="1.5" opacity="0.6"/>')

    for i, r in enumerate(data):
        if not r.get("duration_min"):
            continue
        x = pts[i][0]
        y = pts[i][1]
        c = colors.get(r["conclusion"], "#94a3b8")
        dt_str = times[i].strftime("%Y-%m-%d %H:%M UTC")
        tooltip = f"{dt_str} | {r['conclusion']} | {r['duration_min']:.1f}m | {r['sha']}"
        svg.append(f'<circle cx="{x:.1f}" cy="{y:.1f}" r="8" fill="{c}" opacity="0.2"/>')
        svg.append(f'<circle cx="{x:.1f}" cy="{y:.1f}" r="5" fill="{c}" stroke="#0f172a" stroke-width="1.5">'
                   f'<title>{tooltip}</title></circle>')

    leg_x = ml + 10
    leg_y = mt + plot_h + 42
    for label, color in [("success", "#22c55e"), ("failure", "#f87171"), ("cancelled", "#fbbf24")]:
        svg.append(f'<circle cx="{leg_x+6}" cy="{leg_y}" r="5" fill="{color}"/>')
        svg.append(f'<text x="{leg_x+16}" y="{leg_y+4}" fill="#94a3b8" font-size="11">{label}</text>')
        leg_x += 90

    today = datetime.now(timezone.utc).strftime("%Y-%m-%d")
    svg.append(f'<text x="{width//2}" y="{height-12}" fill="#475569" font-size="9" text-anchor="middle">'
               f"Data: GitHub Checks API → CircleCI Checks  ·  Generated {today}  ·  "
               f"Wall-clock = workflow start→completion</text>")
    svg.append("</svg>")

    with open(out_path, "w") as f:
        f.write("\n".join(svg))
    print(f"SVG written to {out_path}")


def main():
    days = 7
    for i, arg in enumerate(sys.argv[1:]):
        if arg == "--days" and i + 1 < len(sys.argv[1:]):
            days = int(sys.argv[i + 2])

    token = os.environ.get("GITHUB_TOKEN")
    script_dir = os.path.dirname(os.path.abspath(__file__))
    data_path = os.path.join(script_dir, "main-workflow-runtime-data.json")
    svg_path = os.path.join(script_dir, "main-workflow-runtime.svg")

    data = collect_data(days=days, token=token)

    with open(data_path, "w") as f:
        json.dump(data, f, indent=2)
    print(f"Data written to {data_path}")

    make_svg(data, svg_path)

    # Print ASCII summary
    data_sorted = sorted(data, key=lambda r: r.get("started_at") or r["commit_date"])
    full_runs = [r for r in data_sorted if r.get("duration_min") and r["duration_min"] >= 5]
    if full_runs:
        avg = sum(r["duration_min"] for r in full_runs) / len(full_runs)
        max_d = max(r["duration_min"] for r in full_runs)
        by_day = defaultdict(list)
        for r in data_sorted:
            day = (r.get("started_at") or r["commit_date"])[:10]
            by_day[day].append(r)
        print(f"\nDaily averages (wall-clock):")
        for day in sorted(by_day.keys()):
            runs = by_day[day]
            fr = [r for r in runs if r.get("duration_min") and r["duration_min"] >= 5]
            if not fr:
                continue
            day_avg = sum(r["duration_min"] for r in fr) / len(fr)
            conc = Counter(r["conclusion"] for r in runs)
            bar = int(round(day_avg / max_d * 35))
            print(f"  {day}  {'█'*bar:<35}  {day_avg:5.1f}m  {dict(conc)}")


if __name__ == "__main__":
    main()
