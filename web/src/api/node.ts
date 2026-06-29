import { request } from '@/api'

export interface NodeMetrics {
  cpu_percent: number
  mem_percent: number
  disk_percent: number
  tx_bytes?: number
  rx_bytes?: number
}

export interface NodeDTO {
  id: string
  name: string
  type: 'runner' | 'panel'
  status: string
  ip: string
  hostname: string
  os: string
  arch: string
  version: string
  last_seen_at?: string
  url?: string
  remark: string
  metrics: NodeMetrics
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface NodeToken {
  id: string
  token: string
  remark: string
  max_uses: number
  used_count: number
  expires_at?: string
  enabled: boolean
  created_at: string
}

export function getNodes() {
  return request<NodeDTO[]>('/nodes', { method: 'GET' })
}

export function createNode(data: { type: 'runner' | 'panel'; name: string; url?: string; token?: string; remark?: string }) {
  return request<any>('/nodes', {
    method: 'POST',
    body: JSON.stringify(data)
  })
}

export function updateNode(id: string, data: { type: 'runner' | 'panel'; name: string; remark?: string; enabled: boolean }) {
  return request<any>(`/nodes/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data)
  })
}

export function deleteNode(id: string, type: 'runner' | 'panel') {
  return request<void>(`/nodes/${id}?type=${type}`, { method: 'DELETE' })
}

// Token management for Runner
export function getTokens() {
  return request<NodeToken[]>('/nodes/tokens', { method: 'GET' })
}

export function createToken(data: { remark?: string; max_uses?: number; expires_at?: string }) {
  return request<NodeToken>('/nodes/tokens', {
    method: 'POST',
    body: JSON.stringify(data)
  })
}

export function deleteToken(id: string) {
  return request<void>(`/nodes/tokens/${id}`, { method: 'DELETE' })
}

export function getVersion() {
  return request<{ version: string; platforms: { os: string; arch: string; filename: string }[] }>('/agents/version')
}
