import { computed } from 'vue'

import { useTransactionFlowStore } from '@/stores'

const rubFormatter = new Intl.NumberFormat('ru-RU', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})
const litersFormatter = new Intl.NumberFormat('ru-RU', {
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
})

export function useOrderSummary() {
  const store = useTransactionFlowStore()
  const orderSummary = computed(() => store.orderSummary)

  const orderFuelType = computed(() => orderSummary.value.fuelType ?? '—')

  const orderLiters = computed(() =>
    orderSummary.value.liters === null ? '—' : `${litersFormatter.format(orderSummary.value.liters)} л`,
  )

  const orderUnitPrice = computed(() =>
    orderSummary.value.unitPrice === null ? '—' : `${rubFormatter.format(orderSummary.value.unitPrice)} ₽`,
  )

  const orderTotalAmount = computed(() =>
    orderSummary.value.totalAmount === null ? '—' : `${rubFormatter.format(orderSummary.value.totalAmount)} ₽`,
  )

  return {
    orderFuelType,
    orderLiters,
    orderUnitPrice,
    orderTotalAmount,
  }
}
