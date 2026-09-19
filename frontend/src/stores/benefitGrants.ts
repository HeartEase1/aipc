import { defineStore } from 'pinia'
import { ref } from 'vue'
import benefitGrantsAPI, { type UserBenefitGrant } from '@/api/benefitGrants'

const THROTTLE_MS = 5 * 60 * 1000

export const useBenefitGrantStore = defineStore('benefitGrants', () => {
  const currentPopup = ref<UserBenefitGrant | null>(null)
  const queue = ref<UserBenefitGrant[]>([])
  const loading = ref(false)
  const lastFetchTime = ref(0)
  let shownIDs = new Set<number>()

  async function fetchUnread(force = false) {
    const now = Date.now()
    if (!force && lastFetchTime.value > 0 && now - lastFetchTime.value < THROTTLE_MS) return
    lastFetchTime.value = now
    try {
      loading.value = true
      const result = await benefitGrantsAPI.list(1, 20, true)
      for (const item of result.items) {
        if (!shownIDs.has(item.id) && !queue.value.some((queued) => queued.id === item.id)) {
          queue.value.push(item)
        }
      }
      if (!currentPopup.value) showNext()
    } catch (error) {
      lastFetchTime.value = 0
      console.error('Failed to fetch benefit grant notifications:', error)
    } finally {
      loading.value = false
    }
  }

  function showNext() {
    currentPopup.value = queue.value.shift() || null
    if (currentPopup.value) shownIDs.add(currentPopup.value.id)
  }

  async function dismissPopup() {
    if (!currentPopup.value) return
    const id = currentPopup.value.id
    currentPopup.value = null
    try {
      await benefitGrantsAPI.markRead(id)
    } catch (error) {
      shownIDs.delete(id)
      lastFetchTime.value = 0
      console.error('Failed to mark benefit grant notification as read:', error)
    }
    if (queue.value.length > 0) {
      setTimeout(showNext, 300)
    } else {
      await fetchUnread(true)
    }
  }

  function reset() {
    currentPopup.value = null
    queue.value = []
    loading.value = false
    lastFetchTime.value = 0
    shownIDs = new Set<number>()
  }

  return { currentPopup, loading, fetchUnread, dismissPopup, reset }
})
