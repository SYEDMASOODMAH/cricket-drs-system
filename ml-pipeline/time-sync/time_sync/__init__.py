"""Multi-camera time synchronization via audio cross-correlation.

See docs/adr/0006-time-sync-language-split.md: this package owns the
actual sync *math* (FFT-based cross-correlation of two audio signals);
storing a computed offset against a clip lives in
services/media-ingest-gateway (Go). Nothing here talks to that service
directly yet — see this package's README.
"""

from time_sync.audio_correlation import SyncResult, find_offset
from time_sync.audio_extraction import AudioExtractionError, extract_audio, find_offset_from_videos

__all__ = [
    "AudioExtractionError",
    "SyncResult",
    "extract_audio",
    "find_offset",
    "find_offset_from_videos",
]
