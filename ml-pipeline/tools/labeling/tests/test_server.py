"""Tests for the labeling tool's pure logic — frame caching and label
read/write/upsert. The canvas UI itself is verified manually in a browser
per the implementation plan (not practical to unit-test a canvas click
interaction here).

Uses time-sync's existing real video fixture rather than inventing a new
one — this tool doesn't care what's actually in a clip, only that it's a
real, valid video file ffmpeg/ffprobe can process.
"""

from __future__ import annotations

from pathlib import Path

import pytest

from tools.labeling import server

FIXTURE = Path(__file__).parent.parent.parent.parent / "time-sync" / "tests" / "fixtures" / "real_sample_video.mp4"


@pytest.fixture(autouse=True)
def _isolated_dataset_dirs(tmp_path: Path, monkeypatch: pytest.MonkeyPatch) -> None:
    """Points the module's dataset/cache/labels dirs at a tmp_path for
    every test, so tests never touch the user's real dataset/ directory."""
    monkeypatch.setattr(server, "DATASET_DIR", tmp_path)
    monkeypatch.setattr(server, "RAW_FOOTAGE_DIR", tmp_path / "raw-footage")
    monkeypatch.setattr(server, "LABELS_DIR", tmp_path / "labels")
    monkeypatch.setattr(server, "FRAME_CACHE_DIR", tmp_path / ".frame-cache")
    (tmp_path / "raw-footage").mkdir()


def _copy_fixture_clip(tmp_path: Path, name: str = "clip.mp4") -> Path:
    dest = server.RAW_FOOTAGE_DIR / name
    dest.write_bytes(FIXTURE.read_bytes())
    return dest


def test_list_clips_finds_video_files(tmp_path: Path) -> None:
    _copy_fixture_clip(tmp_path)
    (server.RAW_FOOTAGE_DIR / "not-a-video.txt").write_text("ignore me")

    clips = server.list_clips()

    assert [c.name for c in clips] == ["clip.mp4"]


def test_probe_fps_and_duration_matches_real_fixture(tmp_path: Path) -> None:
    clip = _copy_fixture_clip(tmp_path)

    fps, duration = server.probe_fps_and_duration(clip)

    assert fps == pytest.approx(29.97, abs=0.5)
    assert duration == pytest.approx(3.0, abs=1.0)


def test_ensure_frames_extracted_populates_cache_once(tmp_path: Path) -> None:
    clip = _copy_fixture_clip(tmp_path)

    frames_first = server.ensure_frames_extracted(clip)
    cache_dir = server.frame_cache_dir_for(clip)
    mtimes_first = {f: f.stat().st_mtime for f in frames_first}

    frames_second = server.ensure_frames_extracted(clip)

    assert len(frames_first) > 0
    assert frames_first == frames_second
    assert all(f.stat().st_mtime == mtimes_first[f] for f in frames_second), (
        "second call must reuse the cache, not re-extract"
    )
    assert cache_dir.is_dir()


def test_save_and_load_label_round_trips(tmp_path: Path) -> None:
    clip = _copy_fixture_clip(tmp_path)

    server.save_label(clip, fps=30.0, frame_count=90, frame_index=5, label={"x": 10, "y": 20, "w": 24, "h": 24, "visible": True})
    doc = server.load_labels(clip, fps=30.0, frame_count=90)

    assert doc["labels"]["5"] == {"x": 10, "y": 20, "w": 24, "h": 24, "visible": True}
    assert doc["frame_count"] == 90


def test_not_visible_is_distinct_from_unreviewed(tmp_path: Path) -> None:
    clip = _copy_fixture_clip(tmp_path)

    doc_before = server.load_labels(clip, fps=30.0, frame_count=90)
    assert "7" not in doc_before["labels"], "an unreviewed frame must have no entry at all"

    server.save_label(clip, fps=30.0, frame_count=90, frame_index=7, label={"visible": False})
    doc_after = server.load_labels(clip, fps=30.0, frame_count=90)

    assert doc_after["labels"]["7"] == {"visible": False}


def test_clearing_a_label_removes_it_entirely(tmp_path: Path) -> None:
    clip = _copy_fixture_clip(tmp_path)
    server.save_label(clip, fps=30.0, frame_count=90, frame_index=3, label={"x": 1, "y": 2, "w": 3, "h": 4, "visible": True})

    server.save_label(clip, fps=30.0, frame_count=90, frame_index=3, label=None)
    doc = server.load_labels(clip, fps=30.0, frame_count=90)

    assert "3" not in doc["labels"], "clearing a label must return the frame to 'not yet reviewed', not an empty entry"
