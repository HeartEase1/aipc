export const calculateOutputRate = (
  outputTokens: number | null | undefined,
  durationMs: number | null | undefined,
  firstTokenMs: number | null | undefined
): number | null => {
  if (
    outputTokens == null
    || outputTokens <= 0
    || durationMs == null
    || firstTokenMs == null
    || durationMs <= firstTokenMs
  ) {
    return null
  }

  const rate = outputTokens * 1000 / (durationMs - firstTokenMs)
  return Number.isFinite(rate) ? rate : null
}

export const formatOutputRate = (
  outputTokens: number | null | undefined,
  durationMs: number | null | undefined,
  firstTokenMs: number | null | undefined
): string => {
  const rate = calculateOutputRate(outputTokens, durationMs, firstTokenMs)
  return rate == null ? '-' : `${rate.toFixed(1)} t/s`
}
