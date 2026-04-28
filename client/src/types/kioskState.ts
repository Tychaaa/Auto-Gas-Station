// Состояние киоска которое сервер отдает публично на /api/v1/kiosk/state
// Используется и киоск-фронтом (для оверлея), и админкой (для индикатора)
export interface KioskState {
  maintenance: boolean
  reason: string
  updatedAt: string
}
