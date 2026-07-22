export const calculateCacheHitRate = (
  inputTokens: number | null | undefined,
  cacheReadTokens: number | null | undefined
): number => {
  const input = Math.max(inputTokens ?? 0, 0)
  const cacheRead = Math.max(cacheReadTokens ?? 0, 0)
  const cacheableInput = input + cacheRead

  return cacheableInput > 0 ? (cacheRead / cacheableInput) * 100 : 0
}
