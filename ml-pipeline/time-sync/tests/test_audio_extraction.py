"""Tests for real audio extraction (time_sync/audio_extraction.py).

real_sample_video.mp4 is a ~3s/320px-wide/low-bitrate trim of a real DJI
Action 5 Pro recording (the same hardware edge-agent targets) — only the
audio content matters here, so video quality was deliberately minimized to
keep this a small, real, checked-in fixture rather than a synthetic one.
"""

from __future__ import annotations

import subprocess
from pathlib import Path

import numpy as np
import pytest
from time_sync.audio_extraction import AudioExtractionError, extract_audio, find_offset_from_videos

FIXTURE_PATH = Path(__file__).parent / "fixtures" / "real_sample_video.mp4"


def test_extract_audio_from_real_fixture() -> None:
    samples, sample_rate = extract_audio(FIXTURE_PATH, sample_rate=48000)

    assert sample_rate == 48000
    assert samples.dtype == np.int16
    assert len(samples) > sample_rate * 2, "expected at least 2 seconds of audio from a ~3s fixture"
    assert np.std(samples) > 20, (
        f"expected real captured audio to have meaningful amplitude variance, got std={np.std(samples)}"
    )


def test_extract_audio_missing_file_raises() -> None:
    with pytest.raises(AudioExtractionError):
        extract_audio(Path("does-not-exist.mp4"))


def test_extract_audio_invalid_file_raises(tmp_path: Path) -> None:
    not_a_video = tmp_path / "not_a_video.mp4"
    not_a_video.write_text("this is plainly not a video file")

    with pytest.raises(AudioExtractionError):
        extract_audio(not_a_video)


def test_find_offset_from_videos_recovers_shift(tmp_path: Path) -> None:
    """Builds a second, genuinely shifted real video file (ffmpeg's adelay
    filter prepends real silence to a copy of the fixture's audio track,
    pushing its matching content later in its own timeline) and confirms
    find_offset_from_videos recovers that shift starting from two real
    files on disk — not pre-decoded arrays, which is the actual gap this
    module closes.
    """
    shift_ms = 300
    shifted_path = tmp_path / "shifted.mp4"
    subprocess.run(
        [
            "ffmpeg",
            "-y",
            "-v",
            "error",
            "-i",
            str(FIXTURE_PATH),
            "-af",
            f"adelay={shift_ms}:all=1",
            "-c:v",
            "copy",
            str(shifted_path),
        ],
        check=True,
    )

    result = find_offset_from_videos(FIXTURE_PATH, shifted_path, sample_rate=48000)

    # Generous tolerance: AAC's own encode/decode block alignment (~21ms
    # frames) and encoder priming samples mean a real re-encoded shift
    # isn't recovered to sample-exact precision the way the synthetic,
    # purely-numpy shifts in test_real_audio.py are.
    assert result.offset_ms == pytest.approx(shift_ms, abs=50)
    assert result.correlation_score > 0.3
