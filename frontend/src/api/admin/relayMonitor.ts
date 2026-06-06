/**
 * Admin Relay Monitor (中转站监控) API endpoints.
 * Tracks external relay stations' (sub2api / newapi) group rate multipliers
 * and records up/down changes.
 */

import { apiClient } from '../client'

export type RelaySystem = 'sub2api' | 'newapi'
export type RateDirection = 'up' | 'down'

export interface RelayMonitor {
  id: number
  name: string
  system: RelaySystem
  base_url: string
  vendor: string
  auth_account: string
  credential_masked: string
  has_credential: boolean
  credential_decrypt_failed: boolean
  watched_groups: string[]
  enabled: boolean
  interval_seconds: number
  last_checked_at: string | null
  last_error: string
  created_at: string
  updated_at: string
}

export interface RelayGroupRate {
  group_name: string
  rate: number
}

export interface RelayRateChange {
  id: number
  monitor_id: number
  site: string
  system: RelaySystem
  vendor: string
  group_name: string
  old_rate: number
  new_rate: number
  direction: RateDirection
  content: string
  detected_at: string
}

export interface RelayProbeResult {
  rates: RelayGroupRate[]
  changes: RelayRateChange[]
}

export interface RelaySummary {
  up_count: number
  down_count: number
}

export interface RelayOverviewRow {
  monitor_id: number
  site: string
  system: RelaySystem
  vendor: string
  group_name: string
  current_rate: number
  has_change: boolean
  old_rate: number
  new_rate: number
  direction: RateDirection | ''
  changed_at: string | null
}

export interface ListParams {
  page?: number
  page_size?: number
  system?: RelaySystem
  enabled?: boolean
  search?: string
}

export interface ListResponse {
  items: RelayMonitor[]
  total: number
  page: number
  page_size: number
  pages: number
}

export interface CreateParams {
  name: string
  system: RelaySystem
  base_url: string
  vendor?: string
  auth_account?: string
  credential?: string
  watched_groups?: string[]
  enabled?: boolean
  interval_seconds?: number
}

// Update: credential 空串 = 不修改
export type UpdateParams = Partial<CreateParams>

export interface ChangesParams {
  page?: number
  page_size?: number
  monitor_id?: number
  direction?: RateDirection
  search?: string
}

export interface ChangesResponse {
  items: RelayRateChange[]
  total: number
  page: number
  page_size: number
  pages: number
}

export interface FetchGroupsParams {
  system: RelaySystem
  base_url: string
  auth_account?: string
  credential?: string
  monitor_id?: number
}

export interface ProbeAllItem {
  monitor_id: number
  name: string
  changes: number
  error?: string
}

export interface ProbeAllResponse {
  probed: number
  results: ProbeAllItem[]
}

/** List relay monitors with pagination and filters. */
export async function list(
  params: ListParams = {},
  options?: { signal?: AbortSignal }
): Promise<ListResponse> {
  const { data } = await apiClient.get<ListResponse>('/admin/relay-monitors', {
    params,
    signal: options?.signal,
  })
  return data
}

/** Get a relay monitor by ID. */
export async function get(id: number): Promise<RelayMonitor> {
  const { data } = await apiClient.get<RelayMonitor>(`/admin/relay-monitors/${id}`)
  return data
}

/** Create a new relay monitor. */
export async function create(params: CreateParams): Promise<RelayMonitor> {
  const { data } = await apiClient.post<RelayMonitor>('/admin/relay-monitors', params)
  return data
}

/** Update an existing relay monitor. credential empty = keep current. */
export async function update(id: number, params: UpdateParams): Promise<RelayMonitor> {
  const { data } = await apiClient.put<RelayMonitor>(`/admin/relay-monitors/${id}`, params)
  return data
}

/** Delete a relay monitor. */
export async function del(id: number): Promise<void> {
  await apiClient.delete(`/admin/relay-monitors/${id}`)
}

/** Probe a single monitor now; returns current rates + detected changes. */
export async function probe(id: number): Promise<RelayProbeResult> {
  const { data } = await apiClient.post<RelayProbeResult>(`/admin/relay-monitors/${id}/probe`)
  return data
}

/** Probe all enabled monitors now. */
export async function probeAll(): Promise<ProbeAllResponse> {
  const { data } = await apiClient.post<ProbeAllResponse>('/admin/relay-monitors/probe-all')
  return data
}

/** Fetch the target site's full group list + current rates (no persistence). */
export async function fetchGroups(params: FetchGroupsParams): Promise<RelayGroupRate[]> {
  const { data } = await apiClient.post<RelayGroupRate[]>('/admin/relay-monitors/fetch-groups', params)
  return data
}

/** List rate-change history (up/down announcements). */
export async function listChanges(
  params: ChangesParams = {},
  options?: { signal?: AbortSignal }
): Promise<ChangesResponse> {
  const { data } = await apiClient.get<ChangesResponse>('/admin/relay-monitors/changes', {
    params,
    signal: options?.signal,
  })
  return data
}

/** Get up/down announcement counts (top summary cards). */
export async function summary(search?: string): Promise<RelaySummary> {
  const { data } = await apiClient.get<RelaySummary>('/admin/relay-monitors/summary', {
    params: search ? { search } : {},
  })
  return data
}

/** Current rate of every watched group, changed ones first with delta. */
export async function overview(search?: string): Promise<RelayOverviewRow[]> {
  const { data } = await apiClient.get<RelayOverviewRow[]>('/admin/relay-monitors/overview', {
    params: search ? { search } : {},
  })
  return data
}

export const relayMonitorAPI = {
  list,
  get,
  create,
  update,
  del,
  probe,
  probeAll,
  fetchGroups,
  listChanges,
  summary,
  overview,
}

export default relayMonitorAPI
