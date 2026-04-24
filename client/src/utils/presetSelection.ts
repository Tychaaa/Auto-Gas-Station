import type { PresetSelection } from '@/types'

function parsePositiveNumber(raw: string): number | null {
  const parsed = Number(raw.replace(',', '.'))
  return Number.isFinite(parsed) && parsed > 0 ? parsed : null
}

export function fromPresetString(preset: string): PresetSelection {
  const normalized = preset.trim()
  if (!normalized) {
    return null
  }

  if (normalized.startsWith('liters_')) {
    const value = parsePositiveNumber(normalized.slice('liters_'.length))
    return value === null ? null : { kind: 'liters', value }
  }

  if (normalized.startsWith('fast_')) {
    const value = parsePositiveNumber(normalized.slice('fast_'.length))
    return value === null ? null : { kind: 'amount', value }
  }

  return null
}

export function toPresetString(selection: PresetSelection): string {
  if (!selection) {
    return ''
  }
  return selection.kind === 'liters' ? `liters_${selection.value}` : `fast_${selection.value}`
}
