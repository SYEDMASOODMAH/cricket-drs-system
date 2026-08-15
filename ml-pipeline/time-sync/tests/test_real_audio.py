"""Time-sync tests against real captured audio, not synthetic noise.

docs/adr/0006's "Revisit if" clause named this exact moment: once real
audio capture exists (edge-agent's internal/capture.OpenMicrophone), the
algorithm should be proven against real signal characteristics, not just
np.random.default_rng(...).standard_normal(...). fixtures/real_sample.wav
is a 2.5s mono clip trimmed from a real capture against the DJI Action 5
Pro's microphone during edge-agent's manual verification (see the
implementation plan) — ambient room audio, not a synthetic tone.

The ground-truth offset is still synthetic (nothing else would give a
known-correct answer to check against), but the base signal being shifted
is genuine captured audio — a meaningfully stronger validation than pure
Gaussian noise, which has none of a real signal's structure (transients,
correlated harmonics, silence gaps) that could in principle trip up a
correlation-based algorithm differently than noise does.
"""

from __future__ import annotations

import wave
from pathlib import Path

import numpy as np
import pytest
from time_sync.audio_correlation import find_offset

FIXTURE_PATH = Path(__file__).parent / "fixtures" / "real_sample.wav"


def _load_wav_mono(path: Path) -> tuple[np.ndarray, int]:
    """Load a 16-bit mono WAV via the stdlib wave module — no new
    dependency, mirroring internal/wav's choice on the Go side (see
    docs/adr/0006 and edge-agent's README for why WAV was chosen as the
    cross-language interop format).
    """
    with wave.open(str(path), "rb") as f:
        if f.getnchannels() != 1 or f.getsampwidth() != 2:
            raise ValueError("expected a 16-bit mono WAV fixture")
        sample_rate = f.getframerate()
        raw = f.readframes(f.getnframes())
    samples = np.frombuffer(raw, dtype=np.int16).astype(np.float64)
    return samples, sample_rate


def _shifted_real_signal(signal: np.ndarray, offset_samples: int, noise_std: float, rng: np.random.Generator) -> np.ndarray:
    """Same construction as test_audio_correlation.py's _shifted_signal,
    just applied to real captured audio instead of synthetic noise.
    """
    n = len(signal)
    shifted = np.zeros_like(signal)
    if offset_samples >= 0:
        shifted[offset_samples:] = signal[: n - offset_samples]
    else:
        shifted[: n + offset_samples] = signal[-offset_samples:]
    return shifted + rng.normal(0, noise_std, size=shifted.shape)


def test_fixture_exists_and_has_real_variance() -> None:
    """Sanity check on the fixture itself before trusting tests built on
    top of it — confirms it's genuine captured audio, not silence or a
    corrupt/empty file.
    """
    samples, sample_rate = _load_wav_mono(FIXTURE_PATH)
    assert sample_rate > 0
    assert len(samples) > sample_rate, "expected at least 1 second of audio"
    assert np.std(samples) > 20, (
        f"expected real captured audio to have meaningful amplitude variance, got std={np.std(samples)}"
    )


def test_find_offset_recovers_known_shift_on_real_audio() -> None:
    reference, sample_rate = _load_wav_mono(FIXTURE_PATH)
    rng = np.random.default_rng(42)
    shift_samples = 300
    target = _shifted_real_signal(reference, offset_samples=shift_samples, noise_std=0.0, rng=rng)

    result = find_offset(reference, target, sample_rate)

    expected_ms = shift_samples / sample_rate * 1000.0
    assert result.offset_ms == pytest.approx(expected_ms)
    assert result.correlation_score > 0.99, (
        f"expected a near-perfect match on a clean shifted copy of real audio, got score {result.correlation_score}"
    )


def test_find_offset_recovers_known_negative_shift_on_real_audio() -> None:
    reference, sample_rate = _load_wav_mono(FIXTURE_PATH)
    rng = np.random.default_rng(43)
    shift_samples = -450
    target = _shifted_real_signal(reference, offset_samples=shift_samples, noise_std=0.0, rng=rng)

    result = find_offset(reference, target, sample_rate)

    expected_ms = shift_samples / sample_rate * 1000.0
    assert result.offset_ms == pytest.approx(expected_ms)
    assert result.correlation_score > 0.99


def test_find_offset_robust_to_noise_on_real_audio() -> None:
    reference, sample_rate = _load_wav_mono(FIXTURE_PATH)
    rng = np.random.default_rng(7)
    shift_samples = 200
    # noise_std relative to the fixture's own amplitude, not an arbitrary
    # absolute value, since real captured audio's amplitude scale differs
    # from the synthetic unit-variance signals test_audio_correlation.py
    # uses.
    noise_std = float(np.std(reference)) * 0.5
    target = _shifted_real_signal(reference, offset_samples=shift_samples, noise_std=noise_std, rng=rng)

    result = find_offset(reference, target, sample_rate)

    expected_ms = shift_samples / sample_rate * 1000.0
    assert result.offset_ms == pytest.approx(expected_ms)
    # Noise at half the signal's own amplitude should still leave a
    # clearly detectable match, just less confident than the clean case.
    assert 0.5 < result.correlation_score < 0.99, (
        f"expected a confident but reduced score under noise, got {result.correlation_score}"
    )
