"""FFT-based cross-correlation for finding the time offset between two
cameras' audio tracks — the "software/audio-fingerprint sync" approach
docs/phases.md's Phase 2 names for the accessible tier (broadcast tier
uses hardware genlock instead, out of scope here).

This is the actual computation docs/adr/0006 places in ml-pipeline rather
than services/media-ingest-gateway (Go): given a reference camera's audio
and a second camera's audio of the same event, recover how many samples
the second lags (or leads) the first by. The Go service stores this
function's output; it never computes it.
"""

from __future__ import annotations

from dataclasses import dataclass

import numpy as np

MIN_SAMPLES = 2


@dataclass(frozen=True)
class SyncResult:
    """A recovered time offset plus how confident the correlation is —
    correlation_score is what
    services/media-ingest-gateway/internal/domain/clip.go's
    Clip.SyncConfident() thresholds against.

    offset_ms is signed: a positive value means target lags behind
    reference (target's audio matches reference's audio from offset_ms
    earlier); negative means target leads.
    """

    offset_ms: float
    correlation_score: float


def find_offset(reference: np.ndarray, target: np.ndarray, sample_rate: int) -> SyncResult:
    """Recover the time offset between two audio signals of the same
    event, via normalized FFT cross-correlation.

    reference, target: 1-D arrays of audio samples (mono). They need not
    be the same length. sample_rate: samples per second, shared by both
    (real multi-camera audio would need resampling to a common rate
    first — assumed already done here).
    """
    reference = np.asarray(reference, dtype=np.float64)
    target = np.asarray(target, dtype=np.float64)

    if reference.ndim != 1 or target.ndim != 1:
        raise ValueError("reference and target must be 1-D arrays")
    if len(reference) < MIN_SAMPLES or len(target) < MIN_SAMPLES:
        raise ValueError(f"find_offset requires at least {MIN_SAMPLES} samples in each signal")

    # De-mean so a shared DC offset doesn't dominate the correlation.
    reference = reference - reference.mean()
    target = target - target.mean()

    ref_len, tgt_len = len(reference), len(target)
    full_len = ref_len + tgt_len - 1
    # Zero-pad to a size >= full_len so the circular correlation an FFT
    # naturally computes doesn't wrap real signal into itself — this is
    # what makes it equivalent to a true linear cross-correlation.
    fft_size = 1
    while fft_size < full_len:
        fft_size *= 2

    ref_fft = np.fft.rfft(reference, fft_size)
    tgt_fft = np.fft.rfft(target, fft_size)
    # conj(ref_fft) * tgt_fft, not the other way around: this ordering is
    # what makes lag > 0 mean "target lags reference" under the lag
    # convention below — verified empirically against known synthetic
    # shifts, not just derived by hand (see this package's tests).
    correlation_circular = np.fft.irfft(np.conj(ref_fft) * tgt_fft, fft_size)

    # correlation_circular is indexed 0..fft_size-1 by FFT convention;
    # remap to signed lags via modular indexing (negative lags wrap to
    # the high end of the circular buffer).
    lags = np.arange(-(tgt_len - 1), ref_len)
    correlation = correlation_circular[lags % fft_size]

    best_index = int(np.argmax(correlation))
    best_lag = int(lags[best_index])
    peak = correlation[best_index]

    norm = np.sqrt(np.sum(reference**2) * np.sum(target**2))
    correlation_score = float(peak / norm) if norm > 0 else 0.0

    offset_ms = best_lag / sample_rate * 1000.0
    return SyncResult(offset_ms=offset_ms, correlation_score=correlation_score)
