import { describe, expect, it } from 'vitest'

import { calculateCacheHitRate } from '../cacheHitRate'

describe('calculateCacheHitRate', () => {
  it('calculates cached input as a share of regular and cached input', () => {
    expect(calculateCacheHitRate(20_000, 70_000)).toBeCloseTo(77.7778)
  })

  it('returns zero for missing or negative token counts', () => {
    expect(calculateCacheHitRate(undefined, undefined)).toBe(0)
    expect(calculateCacheHitRate(-100, -50)).toBe(0)
  })
})
