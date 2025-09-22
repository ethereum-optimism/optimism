"""Scoring utilities for calculating test priority based on staleness and age."""

import time
from typing import Optional


def calculate_staleness_days(
    test_commit_ts: Optional[int], contract_commit_ts: Optional[int]
) -> Optional[float]:
    """Calculate staleness in days between test and contract commits.

    Args:
        test_commit_ts: Unix timestamp of test file's last commit.
        contract_commit_ts: Unix timestamp of contract file's last commit.

    Returns:
        Staleness in days (positive if contract is newer), or None if timestamps unavailable.
    """
    if test_commit_ts is not None and contract_commit_ts is not None:
        return (contract_commit_ts - test_commit_ts) / 86400
    return None


def calculate_test_score(
    staleness_days: Optional[float], test_commit_ts: Optional[int]
) -> Optional[float]:
    """Calculate test priority score using two-branch scoring algorithm.

    Args:
        staleness_days: Staleness in days (positive if contract is newer).
        test_commit_ts: Unix timestamp of test file's last commit.

    Returns:
        Priority score (higher means more urgent), or None if cannot calculate.
    """
    now_ts = int(time.time())

    if staleness_days is not None:
        if staleness_days > 0:
            # Case 1: Contract newer than test - use staleness_days
            return staleness_days
        elif test_commit_ts is not None:
            # Case 2: Test up to date or newer - use test age
            return (now_ts - test_commit_ts) / 86400
    elif test_commit_ts is not None:
        # Fallback: only test timestamp available - use test age
        return (now_ts - test_commit_ts) / 86400

    return None
