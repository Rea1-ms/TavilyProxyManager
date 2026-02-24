export type Stats = {
  total_quota: number
  total_used: number
  total_remaining: number
  key_count: number
  active_key_count: number
  today_requests: number
}

export type TimeSeries = {
  granularity: string
  labels: string[]
  series: { name: string; data: number[] }[]
}

export type KeyItem = {
  id: number
  key: string
  alias: string
  total_quota: number
  used_quota: number
  is_active: boolean
  is_invalid: boolean
  last_used_at?: string | null
  created_at?: string
}

export type BatchCreateFailure = {
  key: string
  error: string
}

export type BatchCreateJob = {
  id?: string
  status: "idle" | "running" | "completed" | "error"
  error?: string
  total?: number
  completed?: number
  succeeded?: number
  failed?: number
  failures?: BatchCreateFailure[]
  started_at?: string
  ended_at?: string
}

export type BatchCreateJobStartResponse = {
  job: BatchCreateJob
  already_running: boolean
}

export type LogItem = {
  id: number
  request_id: string
  key_used: number
  key_alias: string
  endpoint: string
  status_code: number
  latency: number
  request_body?: string | null
  request_truncated?: boolean
  response_body?: string | null
  response_truncated?: boolean
  client_ip: string
  created_at: string
}

export type LogStatusCount = {
  status_code: number
  count: number
}

export type DistributedKeyItem = {
  id: number
  name: string
  note: string
  key_prefix: string
  is_active: boolean
  expires_at?: string | null
  rate_limit_per_minute: number
  last_used_at?: string | null
  created_at: string
  total_count: number
  status_2xx: number
  status_4xx: number
  status_5xx: number
}

export type DistributedKeyUsagePoint = {
  date: string
  total_count: number
  status_2xx: number
  status_4xx: number
  status_5xx: number
}

export type DistributedKeyStats = {
  item: Omit<DistributedKeyItem, "total_count" | "status_2xx" | "status_4xx" | "status_5xx">
  totals: {
    total_count: number
    status_2xx: number
    status_4xx: number
    status_5xx: number
  }
  series: DistributedKeyUsagePoint[]
  days: number
}
