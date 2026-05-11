export type ScreenCategory = 'free' | 'confirm' | 'blocked' | 'idle'

const SCREEN_LABELS: Record<string, string> = {
  'fuel-select': 'Выбор топлива',
  'order-select': 'Выбор объёма/суммы',
  'payment-method': 'Выбор способа оплаты',
  'payment-pending': 'Оплата',
  'payment-result': 'Результат оплаты',
  'fueling-progress': 'Налив',
  'fueling-done': 'Завершение',
}

const SCREEN_CATEGORIES: Record<string, ScreenCategory> = {
  'fuel-select': 'free',
  'order-select': 'confirm',
  'payment-method': 'confirm',
  'payment-pending': 'blocked',
  'payment-result': 'blocked',
  'fueling-progress': 'blocked',
  'fueling-done': 'free',
}

export function screenLabel(name: string): string {
  return SCREEN_LABELS[name] ?? 'Нет активной сессии'
}

export function categorize(name: string): ScreenCategory {
  return SCREEN_CATEGORIES[name] ?? 'idle'
}

export function blockedWarningText(name: string): string {
  if (name === 'fueling-progress') return 'Сейчас идёт налив топлива — перевод невозможен'
  return 'Сейчас идёт оплата — перевод невозможен'
}
