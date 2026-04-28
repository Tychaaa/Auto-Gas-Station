import type { KioskState } from '@/types/kioskState'

import { httpGet } from './http'

// Загружает актуальное состояние киоска (режим тех работ и причина)
// Публичная ручка без авторизации — ее пуллит киоск-браузер
export async function getKioskState(): Promise<KioskState> {
  return httpGet<KioskState>('/kiosk/state')
}
