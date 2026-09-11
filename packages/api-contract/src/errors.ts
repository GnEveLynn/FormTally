export interface ApiErrorBody {
  error: {
    code: string
    message: string
    requestId: string
    details?: Array<{ field: string; reason: string }>
  }
}
