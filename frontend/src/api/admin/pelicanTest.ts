import { apiClient, buildApiUrl } from '../client'
import { ADMIN_UI_REQUEST_HEADER } from '../adminUIRequest'
import { refreshAuthTokens } from '../tokenRefresh'

export interface PelicanCapabilities {
  model_id: string
  supported_reasoning_levels: string[]
}

export async function getPelicanCapabilities(accountID: number, modelID: string, signal?: AbortSignal) {
  const { data } = await apiClient.get<PelicanCapabilities>(`/admin/accounts/${accountID}/pelican-test/options`, {
    params: { model_id: modelID }, signal
  })
  return data
}

export async function startPelicanRun(accountID: number, payload: { model_id: string; prompt: string; reasoning_effort: string }, signal: AbortSignal) {
  const authUser = localStorage.getItem('auth_user')
  const token = localStorage.getItem('auth_token')
  const send = (accessToken: string | null) => fetch(buildApiUrl(`/admin/accounts/${accountID}/pelican-test`), {
    method: 'POST',
    headers: { Authorization: `Bearer ${accessToken || ''}`, 'Content-Type': 'application/json', [ADMIN_UI_REQUEST_HEADER]: '1' },
    body: JSON.stringify(payload), signal
  })
  const response = await send(token)
  if (response.status !== 401 || !localStorage.getItem('refresh_token')) return response
  await response.body?.cancel()
  const tokens = await refreshAuthTokens({ failedAccessToken: token })
  if (signal.aborted || localStorage.getItem('auth_user') !== authUser) throw new Error('Test session changed')
  return send(tokens.access_token)
}
