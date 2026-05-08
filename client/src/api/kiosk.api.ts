import type { KioskState } from '@/types/kioskState'

import { httpGet, httpPost } from './http'

export async function getKioskState(): Promise<KioskState> {
  return httpGet<KioskState>('/kiosk/state')
}

export async function reportKioskScreen(screen: string): Promise<KioskState> {
  return httpPost<KioskState>('/kiosk/screen', { screen })
}
