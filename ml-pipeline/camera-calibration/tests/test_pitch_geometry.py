"""Sanity checks on the standard cricket pitch reference points."""

from camera_calibration.pitch_geometry import (
    PITCH_LENGTH_M,
    STUMP_HEIGHT_M,
    reference_points,
    reference_points_array,
)


def test_reference_points_count() -> None:
    points = reference_points()
    assert len(points) == 8, "expected 3 stump bases + 1 stump top at each end"


def test_reference_points_span_pitch_length() -> None:
    points = reference_points()
    xs = [p.xyz[0] for p in points]
    assert max(xs) - min(xs) == PITCH_LENGTH_M


def test_reference_points_include_stump_height_variation() -> None:
    points = reference_points()
    zs = {p.xyz[2] for p in points}
    assert 0.0 in zs
    assert STUMP_HEIGHT_M in zs, "stump-top points give the set height variation solvePnP needs"


def test_reference_points_array_matches_named_points() -> None:
    named = reference_points()
    array = reference_points_array()
    assert array.shape == (8, 3)
    for i, p in enumerate(named):
        assert tuple(array[i]) == p.xyz
