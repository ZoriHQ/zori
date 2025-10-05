export type LoginResponse = {
  access_token: string
  refresh_token: string
  expires_in: number
  account: Account
  organization: Organization
}

export interface Account {
  id: string
  email: string
  first_name: string
  last_name: string
  email_verified: boolean
  created_at: string
  updated_at: string
}

export interface Organization {
  id: string
  name: string
  slug: string
  created_at: string
  updated_at: string
}
