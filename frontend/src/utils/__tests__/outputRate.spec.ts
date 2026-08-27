import { describe, expect, it } from 'vitest'

import {
  calculateGenerationOutputRate,
  calculateOutputRate,
  formatGenerationOutputRate,
  formatOutputRate,
} from '../outputRate'

describe('outputRate', () => {
  it('calculates average output speed over the complete request duration', () => {
    expect(calculateOutputRate(497, 11_610)).toBeCloseTo(42.8079)
    expect(formatOutputRate(497, 11_610)).toBe('42.8 t/s')
  })

  it.each([
    { outputTokens: 0, durationMs: 11_610 },
    { outputTokens: 497, durationMs: null },
    { outputTokens: 497, durationMs: 0 },
    { outputTokens: 497, durationMs: -1 },
  ])('returns no rate for incomplete or invalid timing data', ({ outputTokens, durationMs }) => {
    expect(calculateOutputRate(outputTokens, durationMs)).toBeNull()
    expect(formatOutputRate(outputTokens, durationMs)).toBe('-')
  })

  it('does not amplify timing noise from the first-token timestamp', () => {
    expect(calculateOutputRate(135, 5_230)).toBeCloseTo(25.8126)
    expect(formatOutputRate(135, 5_230)).toBe('25.8 t/s')
  })

  it('keeps the previous post-first-token calculation available for comparison', () => {
    expect(calculateGenerationOutputRate(497, 11_610, 4_980)).toBeCloseTo(74.9623)
    expect(formatGenerationOutputRate(497, 11_610, 4_980)).toBe('75.0 t/s')
  })

  it('returns no legacy rate when first-token timing is unavailable or invalid', () => {
    expect(calculateGenerationOutputRate(135, 5_230, null)).toBeNull()
    expect(calculateGenerationOutputRate(135, 5_230, 5_230)).toBeNull()
    expect(calculateGenerationOutputRate(135, 5_230, -1)).toBeNull()
  })
})
