export type OnboardingStatus = 'profile_required' | 'goal_required' | 'completed'

export interface VerificationResponse {
  verification: {
    requestId: string
    expiresInSeconds: number
    retryAfterSeconds: number
  }
}

export interface SessionResponse {
  session: { expiresAt: string }
  user: {
    id: string
    phoneMasked: string | null
    onboardingStatus: OnboardingStatus
  }
  consents: {
    termsVersion: string
    privacyVersion: string
    aiImageProcessingVersion: string | null
    currentAiImageProcessingVersion: string
  }
}

export interface LoginRequest {
  phone: string
  code: string
  verificationRequestId: string
  agreements: {
    termsVersion: string
    privacyVersion: string
  }
}

export interface WeChatSessionRequest {
  loginCode: string
  agreements: { termsVersion: string; privacyVersion: string }
}

export type WeChatSessionResult = { token: string } & SessionResponse

export interface WeChatPhoneBindingRequest {
  bindingTicket: string
  phoneCode: string
  agreements: {
    termsVersion: string
    privacyVersion: string
  }
}

export type WeChatPhoneBindingResponse = { token: string } & SessionResponse
