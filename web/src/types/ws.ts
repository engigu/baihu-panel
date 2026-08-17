export interface WSMessage {
  type: string
  timestamp: number
  payload: any
}

export interface TaskStatusPayload {
  id: string
  task_id: string
  task_name: string
  status: string
  duration: number
  start_time: string | null
  end_time: string | null
  error?: string | null
}
