"""Extrinsic camera-pose estimation against known pitch geometry.

See docs/adr/0005-camera-calibration-language-split.md: this package owns
the actual calibration *math* (OpenCV pose estimation); registration and
profile storage live in services/camera-calibration (Go). Nothing here
talks to that service directly yet — see this package's README.
"""

from camera_calibration.extrinsic import CameraIntrinsics, PoseEstimate, solve_pose
from camera_calibration.pitch_geometry import PITCH_LENGTH_M, reference_points

__all__ = [
    "CameraIntrinsics",
    "PoseEstimate",
    "solve_pose",
    "PITCH_LENGTH_M",
    "reference_points",
]
