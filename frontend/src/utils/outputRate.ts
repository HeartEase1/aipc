export const calculateOutputRate = (
  outputTokens: number | null | undefined,
  durationMs: number | null | undefined
): number | null => {
  if (
    outputTokens == null
    || outputTokens <= 0
    || durationMs == null
    || durationMs <= 0
  ) {
    return null
  }

  // Use the complete request wall-clock duration. TTFT is measured at event
  // boundaries and is not reliable as a subtraction term across all protocol
  // bridges; subtracting it can turn a few milliseconds of timing noise into
  // an implausibly large rate.
  const rate = outputTokens * 1000 / durationMs
  return Number.isFinite(rate) ? rate : null
}

export const formatOutputRate = (
  outputTokens: number | null | undefined,
  durationMs: number | null | undefined
): string => {
  const rate = calculateOutputRate(outputTokens, durationMs)
  return rate == null ? '-' : `${rate.toFixed(1)} t/s`
}

/**
 * Calculates the previous post-first-token output rate for comparison.
 * This is intentionally separate from calculateOutputRate: first-token timing
 * is event-boundary data and can make this value unstable on bridged streams.
 */
export const calculateGenerationOutputRate = (
  outputTokens: number | null | undefined,
  durationMs: number | null | undefined,
  firstTokenMs: number | null | undefined
): number | null => {
  if (
    outputTokens == null
    || outputTokens <= 0
    || durationMs == null
    || durationMs <= 0
    || firstTokenMs == null
    || firstTokenMs < 0
    || firstTokenMs >= durationMs
  ) {
    return null
  }

  const generationDurationMs = durationMs - firstTokenMs
  const rate = outputTokens * 1000 / generationDurationMs
  return Number.isFinite(rate) ? rate : null
}

export const formatGenerationOutputRate = (
  outputTokens: number | null | undefined,
  durationMs: number | null | undefined,
  firstTokenMs: number | null | undefined
): string => {
  const rate = calculateGenerationOutputRate(outputTokens, durationMs, firstTokenMs)
  return rate == null ? '-' : `${rate.toFixed(1)} t/s`
}
