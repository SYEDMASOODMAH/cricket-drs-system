"""Synthetic-fixture tests for audio-correlation time sync.

No real camera, venue, or audio-decode infrastructure is involved: a
known reference signal is deliberately shifted (and optionally noised) to
build a synthetic "second camera" track with a known ground-truth offset,
and find_offset is checked to recover it. This validates the sync
*algorithm* the way docs/phases.md's Phase 2 testing requirements call
for, without any real hardware.
"""

from __future__ import annotations

import numpy as np
import pytest

from time_sync.audio_correlation import find_offset

SAMPLE_RATE = 8000


def _shifted_signal(signal: np.ndarray, offset_samples: int, noise_std: float, rng: np.random.Generator) -> np.ndarray:
    """Build a "second camera" track: signal shifted so it lags the
    original by offset_samples (negative = leads), with independent noise
    added — the same kind of imperfect real-world audio a second camera's
    microphone would actually pick up.
    """
    n = len(signal)
    shifted = np.zeros_like(signal)
    if offset_samples >= 0:
        shifted[offset_samples:] = signal[: n - offset_samples]
    else:
        shifted[: n + offset_samples] = signal[-offset_samples:]
    return shifted + rng.normal(0, noise_std, size=shifted.shape)


def test_find_offset_recovers_known_positive_shift() -> None:
    rng = np.random.default_rng(42)
    reference = rng.standard_normal(4000)
    target = _shifted_signal(reference, offset_samples=250, noise_std=0.0, rng=rng)

    result = find_offset(reference, target, SAMPLE_RATE)

    expected_ms = 250 / SAMPLE_RATE * 1000.0
    assert result.offset_ms == pytest.approx(expected_ms)
    assert result.correlation_score > 0.9


def test_find_offset_recovers_known_negative_shift() -> None:
    rng = np.random.default_rng(43)
    reference = rng.standard_normal(4000)
    target = _shifted_signal(reference, offset_samples=-150, noise_std=0.0, rng=rng)

    result = find_offset(reference, target, SAMPLE_RATE)

    expected_ms = -150 / SAMPLE_RATE * 1000.0
    assert result.offset_ms == pytest.approx(expected_ms)
    assert result.correlation_score > 0.9


def test_find_offset_robust_to_noise() -> None:
    rng = np.random.default_rng(7)
    reference = rng.standard_normal(4000)
    target = _shifted_signal(reference, offset_samples=250, noise_std=0.5, rng=rng)

    result = find_offset(reference, target, SAMPLE_RATE)

    expected_ms = 250 / SAMPLE_RATE * 1000.0
    assert result.offset_ms == pytest.approx(expected_ms)
    # Noise should reduce confidence versus the clean case, but a real
    # shared signal should still be clearly detectable.
    assert 0.5 < result.correlation_score < 0.99


def test_find_offset_self_correlation_is_zero_and_confident() -> None:
    rng = np.random.default_rng(1)
    reference = rng.standard_normal(2000)

    result = find_offset(reference, reference.copy(), SAMPLE_RATE)

    assert result.offset_ms == pytest.approx(0.0)
    assert result.correlation_score == pytest.approx(1.0, abs=1e-6)


def test_find_offset_independent_signals_score_low() -> None:
    rng = np.random.default_rng(99)
    reference = rng.standard_normal(4000)
    unrelated = rng.standard_normal(4000)

    result = find_offset(reference, unrelated, SAMPLE_RATE)

    assert abs(result.correlation_score) < 0.3, (
        f"expected two independent random signals to correlate weakly, got score {result.correlation_score}"
    )


def test_find_offset_handles_different_lengths() -> None:
    rng = np.random.default_rng(5)
    reference = rng.standard_normal(5000)
    target = _shifted_signal(reference, offset_samples=100, noise_std=0.0, rng=rng)[:3000]

    result = find_offset(reference, target, SAMPLE_RATE)

    expected_ms = 100 / SAMPLE_RATE * 1000.0
    assert result.offset_ms == pytest.approx(expected_ms)


def test_find_offset_rejects_too_few_samples() -> None:
    with pytest.raises(ValueError, match="at least"):
        find_offset(np.array([1.0]), np.array([1.0]), SAMPLE_RATE)


def test_find_offset_rejects_non_1d_input() -> None:
    with pytest.raises(ValueError, match="1-D"):
        find_offset(np.zeros((10, 2)), np.zeros((10, 2)), SAMPLE_RATE)
