export interface ScanRequestBody {
  scanner_name?: string
  target_path?: string
  semgrep_config?: string
}

export interface ProcessingSummary {
  run_id: number
  findings_received: number
  findings_processed: number
  vulnerabilities_created: number
  groups_updated: number
}

export interface GroupRow {
  id: number
  group_key: string
  grouping_rule: string
  severity_max: string
  assets_count: number
  status: string
}

export interface TicketRow {
  group_id: number
  jira_issue_key: string
  jira_issue_url: string
  sync_status: string
  idempotency_key: string
}

export interface PassportResponse {
  scanner_name: string
  scan_target: string
  findings: unknown[]
  processing: ProcessingSummary
  groups: GroupRow[]
  tickets: TicketRow[]
}

export interface SyncRunRow {
  id: number
  source_code: string
  status: string
  items_discovered?: number
  items_processed?: number
  started_at?: string
  finished_at?: string
  error_message?: string
}
