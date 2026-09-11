export type BiologicalSex = 'male' | 'female'
export type ActivityLevel = 'sedentary' | 'light' | 'moderate' | 'high' | 'very_high'
export interface HealthContext { pregnant: boolean; breastfeeding: boolean; clinicalDietRequired: boolean }
export interface ProfileInput { biologicalSex: BiologicalSex; birthDate: string; heightCm: number; weightKg: number; activityLevel: ActivityLevel; timezone: string; healthContext: HealthContext; expectedRevision?: number }
export interface Profile extends ProfileInput { automaticGoalEligible: boolean; revision: number; updatedAt: string }
export interface ProfileResponse { profile: Profile | null }
