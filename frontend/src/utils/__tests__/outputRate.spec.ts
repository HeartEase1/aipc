import { describe, expect, it } from 'vitest'

import { calculateOutputRate, formatOutputRate } from '../outputRate'

describe('outputRate', () => {
  it('calculates generation speed after the first token', () => {
    expect(calculateOutputRate(497, 11_610, 4_980)).toBeCloseTo(74.9623)
    expect(formatOutputRate(497, 11_610, 4_980)).toBe('75.0 t/s')
  })

  it.each([
    { outputTokens: 0, durationMs: 11_610, firstTokenMs: 4_980 },
    { outputTokens: 497, durationMs: 11_610, firstTokenMs: null },
    { outputTokens: 497, durationMs: 4_980, firstTokenMs: 4_980 },
    { outputTokens: 497, durationMs: 4_000, firstTokenMs: 4_980 },
  ])('returns no rate for incomplete or invalid timing data', ({ outputTokens, durationMs, firstTokenMs }) => {
    expect(calculateOutputRate(outputTokens, durationMs, firstTokenMs)).toBeNull()
    expect(formatOutputRate(outputTokens, durationMs, firstTokenMs)).toBe('-')
  })
})
