"""Synthetic-fixture tests for extrinsic pose estimation.

No real camera or venue is involved: a known ground-truth camera pose is
used to project the standard pitch reference points (pitch_geometry.py)
into "observed" 2D image coordinates via OpenCV's own projectPoints, and
solve_pose is checked to recover that same pose from those coordinates.
This is the "calibration accuracy validated against measured pitch
geometry" testing requirement docs/phases.md's Phase 2 names, done without
any hardware.
"""

from __future__ import annotations

import cv2
import numpy as np
import pytest

from camera_calibration.extrinsic import CameraIntrinsics, solve_pose
from camera_calibration.pitch_geometry import reference_points_array

# Matches the placeholder GoPro profile seeded in
# services/camera-calibration/internal/domain/lens_profile.go — reused
# here for narrative consistency, not because the two are code-coupled.
INTRINSICS = CameraIntrinsics(
    focal_length_px_x=1420.0,
    focal_length_px_y=1420.0,
    principal_point_px_x=960.0,
    principal_point_px_y=540.0,
    distortion_coeffs=(0.0, 0.0, 0.0, 0.0, 0.0),
)

# An arbitrary but fixed ground-truth pose. t_z=15 keeps every pitch
# point's camera-frame Z positive regardless of rotation (the points'
# worst-case Euclidean distance from the world origin is ~10.1m, well
# under 15m), so this holds up as a valid "camera can see the pitch" pose
# for any rotation vector, not just this specific one.
TRUE_ROTATION = np.array([np.deg2rad(-70), np.deg2rad(5), np.deg2rad(3)])
TRUE_TRANSLATION = np.array([0.0, 2.0, 15.0])


def _project_true_pose(world_points: np.ndarray) -> np.ndarray:
    image_points, _ = cv2.projectPoints(
        world_points,
        TRUE_ROTATION,
        TRUE_TRANSLATION,
        INTRINSICS.camera_matrix(),
        INTRINSICS.dist_coeffs_array(),
    )
    return image_points.reshape(-1, 2)


def test_solve_pose_recovers_known_ground_truth() -> None:
    world_points = reference_points_array()
    image_points = _project_true_pose(world_points)

    estimate = solve_pose(world_points, image_points, INTRINSICS)

    assert np.allclose(estimate.rotation_vector, TRUE_ROTATION, atol=1e-3)
    assert np.allclose(estimate.translation_vector, TRUE_TRANSLATION, atol=1e-3)
    assert estimate.reprojection_error_px < 0.01, (
        f"expected near-zero reprojection error for noiseless synthetic data, "
        f"got {estimate.reprojection_error_px}"
    )


def test_solve_pose_detects_perturbed_correspondences() -> None:
    world_points = reference_points_array()
    image_points = _project_true_pose(world_points)

    clean_estimate = solve_pose(world_points, image_points, INTRINSICS)

    # Deliberately corrupt two of the eight observed points by 50px, as if
    # a point had been mis-clicked/mis-detected during a real calibration
    # capture, and confirm the accuracy metric actually reflects it.
    perturbed_points = image_points.copy()
    perturbed_points[0] += (50.0, -50.0)
    perturbed_points[4] += (-50.0, 50.0)

    perturbed_estimate = solve_pose(world_points, perturbed_points, INTRINSICS)

    assert perturbed_estimate.reprojection_error_px > clean_estimate.reprojection_error_px + 5.0, (
        "expected perturbed correspondences to produce a detectably worse "
        "reprojection error than the clean baseline"
    )


def test_solve_pose_rejects_mismatched_lengths() -> None:
    world_points = reference_points_array()
    image_points = _project_true_pose(world_points)[:-1]  # one fewer point

    with pytest.raises(ValueError, match="same length"):
        solve_pose(world_points, image_points, INTRINSICS)


def test_solve_pose_rejects_too_few_points() -> None:
    world_points = reference_points_array()[:3]
    image_points = _project_true_pose(reference_points_array())[:3]

    with pytest.raises(ValueError, match="at least"):
        solve_pose(world_points, image_points, INTRINSICS)


def test_camera_intrinsics_matrix_shape() -> None:
    matrix = INTRINSICS.camera_matrix()
    assert matrix.shape == (3, 3)
    assert matrix[2, 2] == 1.0
