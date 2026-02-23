#!/usr/bin/env python3
"""Visualize SDM benchmark JSONL output.

Reads JSONL from --input FILE (or stdin). Produces 3 subplots in a single PNG:
  1. Bar chart: mean refund ratio by category
  2. Grouped bar chart: mean canonical gas vs mean effective gas
  3. Box plot: refund ratio distribution per category
"""

import argparse
import json
import sys
from collections import defaultdict

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import matplotlib.ticker as ticker


def load_records(f):
    summaries = {}
    tx_ratios = defaultdict(list)
    for line in f:
        line = line.strip()
        if not line:
            continue
        rec = json.loads(line)
        if rec["type"] == "summary":
            summaries[rec["category"]] = rec
        elif rec["type"] == "tx":
            tx_ratios[rec["category"]].append(rec["refund_ratio"])
    return summaries, tx_ratios


def plot(summaries, tx_ratios, output_path):
    categories = sorted(summaries.keys())
    if not categories:
        print("No summary records found.", file=sys.stderr)
        sys.exit(1)

    fig, axes = plt.subplots(1, 3, figsize=(18, 6))
    fig.suptitle("SDM OPGas Benchmark", fontsize=14, fontweight="bold")

    colors = {
        "eoa_transfer": "#4C72B0",
        "compute_heavy": "#55A868",
        "event_emitter": "#C44E52",
        "state_bloat": "#8172B2",
    }
    bar_colors = [colors.get(c, "#999999") for c in categories]

    # 1. Mean refund ratio by category
    ax = axes[0]
    mean_ratios = [summaries[c]["mean_ratio"] for c in categories]
    bars = ax.bar(categories, mean_ratios, color=bar_colors, edgecolor="black", linewidth=0.5)
    ax.set_title("Mean Refund Ratio by Category")
    ax.set_ylabel("Refund Ratio (OPGasRefund / GasUsed)")
    ax.set_ylim(0, max(mean_ratios) * 1.3 if max(mean_ratios) > 0 else 1.0)
    ax.yaxis.set_major_formatter(ticker.PercentFormatter(xmax=1.0))
    for bar, val in zip(bars, mean_ratios):
        ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height() + 0.005,
                f"{val:.1%}", ha="center", va="bottom", fontsize=9)
    ax.tick_params(axis="x", rotation=20)

    # 2. Grouped bar: canonical vs effective gas
    ax = axes[1]
    import numpy as np
    x = np.arange(len(categories))
    width = 0.35
    canonical = [summaries[c]["mean_canonical"] for c in categories]
    effective = [summaries[c]["mean_effective"] for c in categories]
    bars1 = ax.bar(x - width / 2, canonical, width, label="Canonical Gas", color="#4C72B0", edgecolor="black", linewidth=0.5)
    bars2 = ax.bar(x + width / 2, effective, width, label="Effective Gas", color="#55A868", edgecolor="black", linewidth=0.5)
    ax.set_title("Mean Canonical vs Effective Gas")
    ax.set_ylabel("Gas")
    ax.set_xticks(x)
    ax.set_xticklabels(categories, rotation=20)
    ax.legend()
    ax.yaxis.set_major_formatter(ticker.FuncFormatter(lambda v, _: f"{v:,.0f}"))

    # 3. Box plot of refund ratio distribution
    ax = axes[2]
    data = [tx_ratios.get(c, []) for c in categories]
    bp = ax.boxplot(data, tick_labels=categories, patch_artist=True, notch=True,
                    medianprops=dict(color="black", linewidth=1.5))
    for patch, color in zip(bp["boxes"], bar_colors):
        patch.set_facecolor(color)
        patch.set_alpha(0.7)
    ax.set_title("Refund Ratio Distribution")
    ax.set_ylabel("Refund Ratio")
    ax.yaxis.set_major_formatter(ticker.PercentFormatter(xmax=1.0))
    ax.tick_params(axis="x", rotation=20)

    plt.tight_layout()
    plt.savefig(output_path, dpi=150, bbox_inches="tight")
    print(f"Saved to {output_path}")


def main():
    parser = argparse.ArgumentParser(description="Visualize SDM benchmark results")
    parser.add_argument("--input", "-i", default="-", help="Input JSONL file (default: stdin)")
    parser.add_argument("--output", "-o", default="sdm_bench_report.png", help="Output PNG path")
    args = parser.parse_args()

    if args.input == "-":
        summaries, tx_ratios = load_records(sys.stdin)
    else:
        with open(args.input) as f:
            summaries, tx_ratios = load_records(f)

    plot(summaries, tx_ratios, args.output)


if __name__ == "__main__":
    main()
