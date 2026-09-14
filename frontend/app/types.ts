export type ReportType = 'regular' | 'hr'

export interface Report {
  id: string
  fileName: string
  reportType: ReportType
}

export interface ResultStats {
  number?: number
  key: string
  city?: string
  employee?: string
  object?: string
  employeeMissing?: boolean
  objectMissing?: boolean
  shows: number
  views: number
  contacts: number
  favorite: number
  promotion: number
  viewersCost: number
  targetViewers: number
  viewWithMessage: number
  lookPhone: number
  ppConversion: number
  pkConversion: number
  avgViewPrice: number
  avgContactPrice: number
  expense: number
  response?: number
  avgResponsePrice?: number
  responseConversion?: number
}

export interface StatsResponse {
  reportId: string
  fileName: string
  reportType: ReportType
  stats: ResultStats[]
}

export interface MultiStatsResponse {
  reports: StatsResponse[]
}

export interface UploadResponse {
  id: string
  fileName: string
  reportType: ReportType
  rows: number
  warnings?: string[]
}

export interface User {
  id: number
  username: string
  role: string
}

export interface AuthUser {
  userId: number
  username: string
  role: string
}
