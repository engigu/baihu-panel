import { request } from '@/api'

export interface InterconnectNode {
  id: string
  name: string
  url: string
  token: string
  remark: string
  created_at: string
  updated_at: string
  status?: string
  metrics?: {
    cpu_percent: number
    mem_percent: number
    disk_percent: number
  }
  last_heartbeat_at?: string
}

export function getNodes() {
  return request<InterconnectNode[]>('/interconnect/nodes', { method: 'GET' })
}

export function createNode(data: Partial<InterconnectNode>) {
  return request<InterconnectNode>('/interconnect/nodes', {
    method: 'POST',
    body: JSON.stringify(data)
  })
}

export function updateNode(id: string, data: Partial<InterconnectNode>) {
  return request<InterconnectNode>(`/interconnect/nodes/${id}`, {
    method: 'PUT',
    body: JSON.stringify(data)
  })
}

export function deleteNode(id: string) {
  return request<void>(`/interconnect/nodes/${id}`, { method: 'DELETE' })
}

export function getNodeStatus(id: string) {
  return request<any>(`/interconnect/nodes/${id}/status`, { method: 'GET' })
}

export function syncScript(node_ids: string[], filename: string, content: string) {
  return request<any[]>('/interconnect/sync/script', {
    method: 'POST',
    body: JSON.stringify({ node_ids, filename, content })
  })
}

export function syncEnv(node_ids: string[], envs: any[]) {
  return request<any[]>('/interconnect/sync/env', {
    method: 'POST',
    body: JSON.stringify({ node_ids, envs })
  })
}

export function syncTask(node_ids: string[], tasks: any[]) {
  return request<any[]>('/interconnect/sync/task', {
    method: 'POST',
    body: JSON.stringify({ node_ids, tasks })
  })
}

export function getChildStatus() {
  return request<{ parent_url: string; parent_token: string; connected: boolean }>('/interconnect/child/status', { method: 'GET' })
}
