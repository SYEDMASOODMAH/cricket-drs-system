"""Multi-camera time synchronization via audio cross-correlation.

See docs/adr/0006-time-sync-language-split.md: this package owns the
actual sync *math* (FFT-based cross-correlation of two audio signals);
storing a computed offset against a clip lives in
services/media-ingest-gateway (Go). Nothing here talks to that service
directly yet — see this package's README.
"""

from time_sync.audio_correlation import SyncResult, find_offset

__all__ = ["SyncResult", "find_offset"]
