import { apiClient } from '../client'

export type CardPlatformType = 'ldxp'
export type CardAuthMode = 'public' | 'token' | 'cookie' | 'push'
export type CardEventType = 'new_product' | 'price_down' | 'price_up' | 'new_low' | 'restock' | 'sold_out' | 'offline' | 'online' | 'changed'

export interface CardPlatformMonitor {
  id: number
  name: string
  platform_type: CardPlatformType
  base_url: string
  shop_url: string
  auth_mode: CardAuthMode
  credential_masked: string
  has_credential: boolean
  credential_decrypt_failed: boolean
  ingest_key: string
  push_mode: boolean
  enabled: boolean
  interval_seconds: number
  fetch_pages: number
  last_checked_at: string | null
  last_error: string
  note: string
  created_at: string
  updated_at: string
}

export interface CardProduct {
  id: number
  monitor_id: number
  platform_name: string
  platform_type: CardPlatformType
  external_product_id: string
  title: string
  merchant: string
  category: string
  image_url: string
  product_url: string
  price: number | null
  cost_price: number | null
  stock: number | null
  sales: number | null
  status: string
  lowest_price: number | null
  first_seen_at: string
  last_seen_at: string
  updated_at: string
  last_event_type?: CardEventType | ''
  last_event_at?: string | null
}

export interface CardPriceEvent {
  id: number
  monitor_id: number
  product_id: number | null
  platform: string
  event_type: CardEventType
  title: string
  old_price: number | null
  new_price: number | null
  old_stock: number | null
  new_stock: number | null
  content: string
  detected_at: string
}

export interface CardSummary {
  platform_count: number
  product_count: number
  price_down: number
  restock: number
  error_count: number
}

export interface ListResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
  pages: number
}

export interface MonitorParams {
  page?: number
  page_size?: number
  platform_type?: CardPlatformType | ''
  enabled?: boolean
  search?: string
}

export interface ProductParams {
  page?: number
  page_size?: number
  search?: string
  monitor_id?: number
  platform_type?: CardPlatformType | ''
  status?: string
  in_stock?: boolean
  sort?: string
}

export interface EventParams {
  page?: number
  page_size?: number
  monitor_id?: number
  event_type?: string
  search?: string
}

export interface SaveMonitorParams {
  name: string
  platform_type?: CardPlatformType
  base_url: string
  shop_url?: string
  auth_mode?: CardAuthMode
  credential?: string
  enabled?: boolean
  interval_seconds?: number
  fetch_pages?: number
  note?: string
}

export async function list(params: MonitorParams = {}): Promise<ListResponse<CardPlatformMonitor>> {
  const { data } = await apiClient.get<ListResponse<CardPlatformMonitor>>('/admin/card-platform-monitors', { params })
  return data
}

export async function create(params: SaveMonitorParams): Promise<CardPlatformMonitor> {
  const { data } = await apiClient.post<CardPlatformMonitor>('/admin/card-platform-monitors', params)
  return data
}

export async function update(id: number, params: Partial<SaveMonitorParams>): Promise<CardPlatformMonitor> {
  const { data } = await apiClient.put<CardPlatformMonitor>(`/admin/card-platform-monitors/${id}`, params)
  return data
}

export async function del(id: number): Promise<void> {
  await apiClient.delete(`/admin/card-platform-monitors/${id}`)
}

export async function probe(id: number): Promise<{ products: CardProduct[]; events: CardPriceEvent[] }> {
  const { data } = await apiClient.post<{ products: CardProduct[]; events: CardPriceEvent[] }>(`/admin/card-platform-monitors/${id}/probe`)
  return data
}

export async function probeAll(): Promise<{ probed: number; results: Array<{ monitor_id: number; name: string; products: number; events: number; error?: string }> }> {
  const { data } = await apiClient.post('/admin/card-platform-monitors/probe-all')
  return data
}

export async function products(params: ProductParams = {}): Promise<ListResponse<CardProduct>> {
  const { data } = await apiClient.get<ListResponse<CardProduct>>('/admin/card-platform-monitors/products', { params })
  return data
}

export async function events(params: EventParams = {}): Promise<ListResponse<CardPriceEvent>> {
  const { data } = await apiClient.get<ListResponse<CardPriceEvent>>('/admin/card-platform-monitors/events', { params })
  return data
}

export async function summary(search?: string): Promise<CardSummary> {
  const { data } = await apiClient.get<CardSummary>('/admin/card-platform-monitors/summary', {
    params: search ? { search } : {},
  })
  return data
}

export async function regenerateKey(id: number): Promise<CardPlatformMonitor> {
  const { data } = await apiClient.post<CardPlatformMonitor>(`/admin/card-platform-monitors/${id}/regenerate-key`)
  return data
}

export default { list, create, update, del, probe, probeAll, products, events, summary, regenerateKey }
