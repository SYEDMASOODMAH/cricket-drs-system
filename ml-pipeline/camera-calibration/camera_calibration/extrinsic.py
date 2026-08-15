"""Extrinsic camera-pose estimation via OpenCV's solvePnP.

This is the actual calibration computation docs/adr/0005 places in
ml-pipeline rather than services/camera-calibration (Go): given a set of
known 3D pitch-geometry points (pitch_geometry.py) and their observed 2D
pixel coordinates in a calibration frame, plus a camera's intrinsic
profile, recover that camera's position and orientation relative to the
pitch. The Go service stores this function's output; it never computes it
(architecture.md Section 7's Python-for-CV/ML boundary).
"""

from __future__ import annotations

from dataclasses import dataclass

import cv2
import numpy as np

MIN_CORRESPONDENCES = 4


@dataclass(frozen=True)
class CameraIntrinsics:
    """A camera's intrinsic profile — mirrors the shape of
    services/camera-calibration/internal/domain/lens_profile.go's
    LensProfile (same fields, no direct code coupling between the two
    languages).
    """

    focal_length_px_x: float
    focal_length_px_y: float
    principal_point_px_x: float
    principal_point_px_y: float
    distortion_coeffs: tuple[float, float, float, float, float]

    def camera_matrix(self) -> np.ndarray:
        return np.array(
            [
                [self.focal_length_px_x, 0.0, self.principal_point_px_x],
                [0.0, self.focal_length_px_y, self.principal_point_px_y],
                [0.0, 0.0, 1.0],
            ],
            dtype=np.float64,
        )

    def dist_coeffs_array(self) -> np.ndarray:
        return np.array(self.distortion_coeffs, dtype=np.float64)


@dataclass(frozen=True)
class PoseEstimate:
    """A recovered camera pose plus how well it explains the observed
    points — reprojection_error_px is what
    services/camera-calibration/internal/domain/calibration.go's
    CalibrationProfile.Valid() thresholds against.
    """

    rotation_vector: np.ndarray
    translation_vector: np.ndarray
    reprojection_error_px: float


def solve_pose(
    world_points: np.ndarray,
    image_points: np.ndarray,
    intrinsics: CameraIntrinsics,
) -> PoseEstimate:
    """Recover a camera's extrinsic pose from known-world-point /
    observed-image-point correspondences.

    world_points: (N, 3) array of 3D points in pitch coordinates
    (pitch_geometry.reference_points_array(), or a subset).
    image_points: (N, 2) array of the same points' observed pixel
    coordinates in a calibration frame, in matching order.
    """
    world_points = np.asarray(world_points, dtype=np.float64)
    image_points = np.asarray(image_points, dtype=np.float64)

    if len(world_points) != len(image_points):
        raise ValueError("world_points and image_points must have the same length")
    if len(world_points) < MIN_CORRESPONDENCES:
        raise ValueError(f"solve_pose requires at least {MIN_CORRESPONDENCES} point correspondences")

    camera_matrix = intrinsics.camera_matrix()
    dist_coeffs = intrinsics.dist_coeffs_array()

    success, rvec, tvec = cv2.solvePnP(
        world_points, image_points, camera_matrix, dist_coeffs, flags=cv2.SOLVEPNP_ITERATIVE
    )
    if not success:
        raise ValueError("solvePnP failed to converge on a pose for the given correspondences")

    reprojected, _ = cv2.projectPoints(world_points, rvec, tvec, camera_matrix, dist_coeffs)
    reprojected = reprojected.reshape(-1, 2)
    reprojection_error_px = float(np.linalg.norm(reprojected - image_points, axis=1).mean())

    return PoseEstimate(
        rotation_vector=rvec.reshape(3),
        translation_vector=tvec.reshape(3),
        reprojection_error_px=reprojection_error_px,
    )
