"""Real audio extraction from a video file via ffmpeg.

Closes the gap this package's README and services/media-ingest-gateway's
README both flagged: find_offset (audio_correlation.py) has only ever
operated on already-decoded 1-D sample arrays, never a real .mp4/.mov file.
Shells out to the ffmpeg binary rather than adding a pip dependency
(ffmpeg-python, PyAV) — reimplementing container demuxing/AAC decoding from
scratch would be a far larger undertaking than this package wants to own,
and ffmpeg is already present on this dev machine and pre-installed on
GitHub Actions' ubuntu-latest runners (this repo's CI), so no new install
step is needed anywhere. This is a genuinely new kind of dependency for
this codebase (an external binary on PATH, not a pip package) — see this
package's README Setup section.
"""

from __future__ import annotations

import subprocess
from pathlib import Path

import numpy as np

from time_sync.audio_correlation import SyncResult, find_offset


class AudioExtractionError(Exception):
    """ffmpeg is missing, the file has no audio stream, or ffmpeg itself
    failed — callers only ever need to catch this one exception type."""


def extract_audio(video_path: Path, sample_rate: int = 48000) -> tuple[np.ndarray, int]:
    """Extract mono PCM audio from a real video file via ffmpeg.

    Returns (samples, sample_rate) where samples is a 1-D int16 array —
    the same dtype tests/test_real_audio.py already uses for the checked-in
    WAV fixture, and what find_offset's float64 cast expects as input.
    ffmpeg does the resampling/downmix itself (-ar/-ac) and the raw PCM is
    piped directly from stdout, no temp WAV file involved.
    """
    cmd = [
        "ffmpeg",
        "-v",
        "error",
        "-i",
        str(video_path),
        "-f",
        "s16le",
        "-acodec",
        "pcm_s16le",
        "-ar",
        str(sample_rate),
        "-ac",
        "1",
        "-",
    ]
    try:
        result = subprocess.run(cmd, capture_output=True, check=False)
    except FileNotFoundError as exc:
        raise AudioExtractionError(
            "ffmpeg not found on PATH — required for real audio extraction, "
            "see ml-pipeline/time-sync/README.md's Setup section"
        ) from exc

    if result.returncode != 0:
        raise AudioExtractionError(
            f"ffmpeg failed on {video_path}: {result.stderr.decode(errors='replace').strip()}"
        )

    samples = np.frombuffer(result.stdout, dtype=np.int16)
    if len(samples) == 0:
        raise AudioExtractionError(f"{video_path} produced no audio samples (no audio stream?)")

    return samples, sample_rate


def find_offset_from_videos(
    reference_path: Path, target_path: Path, sample_rate: int = 48000
) -> SyncResult:
    """extract_audio on both files, then find_offset — the real
    clips-in/offset-out entry point this package's README previously
    described as not existing yet."""
    reference, rate = extract_audio(reference_path, sample_rate)
    target, _ = extract_audio(target_path, sample_rate)
    return find_offset(reference, target, rate)
