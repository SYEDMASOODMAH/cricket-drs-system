"""Standard cricket pitch geometry as 3D reference points.

Used as the world-coordinate side of the point correspondences fed into
solve_pose (extrinsic.py) — the "ad-hoc, pitch-geometry-reference-assisted"
accessible-tier calibration architecture.md Section 1a describes, and the
"22-yard length, stump dimensions" reference points prd.md Section 8 names
for AI-assisted calibration.

Coordinate system: origin at the pitch's center (the midpoint between the
two sets of stumps), X axis running along the pitch's length, Y axis
across its width, Z axis up. Units are meters.
"""

from __future__ import annotations

from dataclasses import dataclass

import numpy as np

# Law 6 dimensions (MCC Laws of Cricket): 22 yards between the two sets of
# stumps; each set of 3 stumps spans 9 inches; a stump stands 28 inches
# above the ground.
PITCH_LENGTH_M = 20.12
STUMP_SET_WIDTH_M = 0.2286
STUMP_HEIGHT_M = 0.711


@dataclass(frozen=True)
class PitchPoint:
    """One named 3D reference point on the pitch."""

    name: str
    xyz: tuple[float, float, float]


def _end_points(end_name: str, x: float) -> list[PitchPoint]:
    half_width = STUMP_SET_WIDTH_M / 2
    return [
        PitchPoint(f"{end_name}_leg_stump_base", (x, -half_width, 0.0)),
        PitchPoint(f"{end_name}_middle_stump_base", (x, 0.0, 0.0)),
        PitchPoint(f"{end_name}_off_stump_base", (x, half_width, 0.0)),
        PitchPoint(f"{end_name}_middle_stump_top", (x, 0.0, STUMP_HEIGHT_M)),
    ]


def reference_points() -> list[PitchPoint]:
    """The 8 well-known pitch points used as calibration references: 3
    stump bases + 1 stump top at each end. The stump tops give the point
    set height variation (not all coplanar at z=0), which meaningfully
    improves solvePnP's pose conditioning versus stump bases alone.
    """
    half_length = PITCH_LENGTH_M / 2
    return _end_points("end_a", -half_length) + _end_points("end_b", half_length)


def reference_points_array() -> np.ndarray:
    """reference_points() as an (N, 3) float64 array, in a fixed order —
    the same order solve_pose's caller must supply matching image_points
    in.
    """
    return np.array([p.xyz for p in reference_points()], dtype=np.float64)
