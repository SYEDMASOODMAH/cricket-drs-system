"""Frame-by-frame ball-labeling tool — a dev-only utility, not a shipped
ml-pipeline component. Turns raw clips in dataset/raw-footage/ into
labeled (frame, ball position) pairs under dataset/labels/, the actual
input a future ball-tracking model would train on. See this package's
README for the label JSON format and how to run it.

Frame extraction and label storage are pure functions (test_server.py
exercises them directly); this module wires them into a small FastAPI
app serving both the JSON API and the static frontend.
"""

from __future__ import annotations

import json
import subprocess
from pathlib import Path
from typing import Any

from fastapi import FastAPI, HTTPException
from fastapi.responses import FileResponse, HTMLResponse
from fastapi.staticfiles import StaticFiles

# All paths resolve relative to this file, not the process's current
# working directory — the process may be launched from the project root
# (per .claude/launch.json's convention for every other service) or from
# ml-pipeline/ directly; both must work the same way.
TOOL_DIR = Path(__file__).resolve().parent
STATIC_DIR = TOOL_DIR / "static"
DATASET_DIR = TOOL_DIR.parent.parent / "dataset"
RAW_FOOTAGE_DIR = DATASET_DIR / "raw-footage"
LABELS_DIR = DATASET_DIR / "labels"
FRAME_CACHE_DIR = DATASET_DIR / ".frame-cache"

VIDEO_EXTENSIONS = {".mp4", ".mov", ".lrf", ".avi", ".mkv", ".m4v"}


def list_clips() -> list[Path]:
    """Every video file directly under raw-footage/, sorted for a stable
    clip list across requests."""
    if not RAW_FOOTAGE_DIR.is_dir():
        return []
    return sorted(p for p in RAW_FOOTAGE_DIR.iterdir() if p.suffix.lower() in VIDEO_EXTENSIONS)


def probe_fps_and_duration(clip_path: Path) -> tuple[float, float]:
    """ffprobe's own r_frame_rate/duration for a clip — used once per clip
    to decide how many frames to extract into the cache."""
    result = subprocess.run(
        [
            "ffprobe",
            "-v",
            "error",
            "-select_streams",
            "v:0",
            "-show_entries",
            "stream=r_frame_rate",
            "-show_entries",
            "format=duration",
            "-of",
            "json",
            str(clip_path),
        ],
        capture_output=True,
        check=True,
    )
    data = json.loads(result.stdout)
    num, den = data["streams"][0]["r_frame_rate"].split("/")
    fps = float(num) / float(den)
    duration = float(data["format"]["duration"])
    return fps, duration


def frame_cache_dir_for(clip_path: Path) -> Path:
    return FRAME_CACHE_DIR / clip_path.stem


def ensure_frames_extracted(clip_path: Path) -> list[Path]:
    """Extracts every frame of clip_path into its cache dir on first call;
    later calls just return the already-cached frame list. Short clips
    (confirmed earlier: a ~20min clip's worth of frames extracts in well
    under a second for a few seconds of footage) make "extract everything
    up front" simpler and more robust than seeking to individual frames on
    demand, which is imprecise with variable-frame-rate sources.
    """
    cache_dir = frame_cache_dir_for(clip_path)
    existing = sorted(cache_dir.glob("frame_*.jpg")) if cache_dir.is_dir() else []
    if existing:
        return existing

    cache_dir.mkdir(parents=True, exist_ok=True)
    fps, _ = probe_fps_and_duration(clip_path)
    subprocess.run(
        [
            "ffmpeg",
            "-y",
            "-v",
            "error",
            "-i",
            str(clip_path),
            "-vf",
            f"fps={fps}",
            str(cache_dir / "frame_%05d.jpg"),
        ],
        check=True,
    )
    return sorted(cache_dir.glob("frame_*.jpg"))


def labels_path_for(clip_path: Path) -> Path:
    return LABELS_DIR / f"{clip_path.stem}.json"


def load_labels(clip_path: Path, fps: float, frame_count: int) -> dict[str, Any]:
    """Returns the clip's label document, creating a fresh empty one
    in-memory (not yet written to disk) if none exists yet."""
    path = labels_path_for(clip_path)
    if path.is_file():
        doc: dict[str, Any] = json.loads(path.read_text())
        return doc
    return {
        "clip": clip_path.name,
        "fps": fps,
        "frame_count": frame_count,
        "labels": {},
    }


def save_label(clip_path: Path, fps: float, frame_count: int, frame_index: int, label: dict[str, Any] | None) -> dict[str, Any]:
    """Upserts one frame's label (or removes it, if label is None) and
    writes the whole document back — simple and correct for a single local
    user labeling one frame at a time; no concurrent-write concern (see
    the implementation plan's "Explicitly deferred" section)."""
    doc = load_labels(clip_path, fps, frame_count)
    key = str(frame_index)
    if label is None:
        doc["labels"].pop(key, None)
    else:
        doc["labels"][key] = label

    LABELS_DIR.mkdir(parents=True, exist_ok=True)
    labels_path_for(clip_path).write_text(json.dumps(doc, indent=2))
    return doc


app = FastAPI()


def _find_clip(clip_name: str) -> Path:
    clip_path = RAW_FOOTAGE_DIR / clip_name
    if not clip_path.is_file():
        raise HTTPException(status_code=404, detail=f"clip not found: {clip_name}")
    return clip_path


@app.get("/api/clips")
def api_list_clips() -> list[dict[str, Any]]:
    out = []
    for clip_path in list_clips():
        fps, duration = probe_fps_and_duration(clip_path)
        frame_count = round(fps * duration)
        doc = load_labels(clip_path, fps, frame_count)
        reviewed = len(doc["labels"])
        out.append(
            {
                "name": clip_path.name,
                "frame_count": frame_count,
                "fps": fps,
                "reviewed": reviewed,
            }
        )
    return out


@app.get("/api/clips/{clip_name}/labels")
def api_get_labels(clip_name: str) -> dict[str, Any]:
    clip_path = _find_clip(clip_name)
    fps, duration = probe_fps_and_duration(clip_path)
    frame_count = round(fps * duration)
    return load_labels(clip_path, fps, frame_count)


@app.post("/api/clips/{clip_name}/labels/{frame_index}")
def api_save_label(clip_name: str, frame_index: int, label: dict[str, Any] | None = None) -> dict[str, Any]:
    clip_path = _find_clip(clip_name)
    fps, duration = probe_fps_and_duration(clip_path)
    frame_count = round(fps * duration)
    return save_label(clip_path, fps, frame_count, frame_index, label)


@app.get("/api/clips/{clip_name}/frames/{frame_index}")
def api_get_frame(clip_name: str, frame_index: int) -> FileResponse:
    clip_path = _find_clip(clip_name)
    frames = ensure_frames_extracted(clip_path)
    if frame_index < 1 or frame_index > len(frames):
        raise HTTPException(status_code=404, detail=f"frame {frame_index} out of range (1..{len(frames)})")
    return FileResponse(frames[frame_index - 1])


@app.get("/")
def index() -> HTMLResponse:
    return HTMLResponse((STATIC_DIR / "index.html").read_text())


app.mount("/static", StaticFiles(directory=str(STATIC_DIR)), name="static")
