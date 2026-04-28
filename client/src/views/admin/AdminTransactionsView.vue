<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

import { listAdminTransactions, type AdminTransactionView } from '@/api/admin.api'

const STATUS_LABELS: Record<string, string> = {
  selection: 'Выбор',
  payment_pending: 'Ожидание оплаты',
  paid: 'Оплачено',
  fueling: 'Отпуск',
  fiscalizing: 'Фискализация',
  completed: 'Завершено',
  failed: 'Ошибка',
}

const STATUS_BADGE_CLASSES: Record<string, string> = {
  completed: 'bg-fuel-lime/20 text-fuel-forest',
  failed: 'bg-red-100 text-red-700',
  payment_pending: 'bg-amber-100 text-amber-700',
  paid: 'bg-sky-100 text-sky-700',
  fueling: 'bg-sky-100 text-sky-700',
  fiscalizing: 'bg-sky-100 text-sky-700',
  selection: 'bg-fuel-olive/20 text-fuel-forest',
}

const transactions = ref<AdminTransactionView[]>([])
const isLoading = ref(false)
const loadError = ref<string | null>(null)

const statusFilter = ref<string>('all')

const availableStatuses = computed(() => {
  const set = new Set<string>()
  for (const tx of transactions.value) {
    set.add(tx.status)
  }
  return Array.from(set).sort()
})

const filteredTransactions = computed(() => {
  if (statusFilter.value === 'all') {
    return transactions.value
  }
  return transactions.value.filter((tx) => tx.status === statusFilter.value)
})

function formatTimestamp(iso: string): string {
  if (!iso) {
    return '—'
  }
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) {
    return iso
  }
  return date.toLocaleString('ru-RU', { hour12: false })
}

function formatAmount(rub: number): string {
  return rub.toFixed(2)
}

function formatLiters(liters: number): string {
  return liters > 0 ? liters.toFixed(2) : '—'
}

function statusLabel(status: string): string {
  return STATUS_LABELS[status] ?? status
}

function statusBadgeClass(status: string): string {
  return STATUS_BADGE_CLASSES[status] ?? 'bg-fuel-olive/15 text-fuel-forest'
}

async function refresh(): Promise<void> {
  isLoading.value = true
  loadError.value = null
  try {
    transactions.value = await listAdminTransactions()
  } catch (error) {
    loadError.value =
      error instanceof Error
        ? error.message
        : 'Не удалось загрузить список транзакций. Проверьте авторизацию.'
    transactions.value = []
  } finally {
    isLoading.value = false
  }
}

onMounted(() => {
  void refresh()
})
</script>

<template>
  <section class="flex flex-col gap-6">
    <div class="flex items-center justify-between gap-4">
      <div>
        <h3 class="font-rubik font-semibold text-2xl text-fuel-forest">
          Транзакции
        </h3>
        <p class="font-karla text-sm text-fuel-olive mt-1">
          Сейчас сервер отдает демонстрационный набор. Персистентный журнал появится отдельной итерацией.
        </p>
      </div>

      <div class="flex items-center gap-3">
        <label class="font-karla text-sm text-fuel-forest flex items-center gap-2">
          Фильтр:
          <select
            v-model="statusFilter"
            class="rounded-lg border border-fuel-olive/40 bg-white px-3 py-2
                   font-karla text-sm text-fuel-forest
                   focus:outline-none focus:ring-2 focus:ring-fuel-lime focus:border-fuel-lime"
          >
            <option value="all">Все статусы</option>
            <option v-for="status in availableStatuses" :key="status" :value="status">
              {{ statusLabel(status) }}
            </option>
          </select>
        </label>
        <button
          type="button"
          class="font-rubik font-medium text-sm px-4 py-2 rounded-lg bg-fuel-forest text-white hover:bg-fuel-olive transition-colors"
          @click="refresh"
        >
          Обновить
        </button>
      </div>
    </div>

    <div class="bg-white rounded-2xl border border-fuel-olive/20 shadow-sm overflow-hidden">
      <p v-if="isLoading" class="font-karla text-sm text-fuel-olive p-6">
        Загружаем список транзакций...
      </p>
      <p v-else-if="loadError" class="font-karla text-sm text-red-600 p-6">
        {{ loadError }}
      </p>
      <p v-else-if="filteredTransactions.length === 0" class="font-karla text-sm text-fuel-olive p-6">
        Нет транзакций по выбранному фильтру
      </p>
      <table v-else class="w-full">
        <thead class="bg-fuel-cream/60 border-b border-fuel-olive/25">
          <tr>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-3 px-4 text-left">ID</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-3 px-4 text-left">Время</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-3 px-4 text-left">Топливо</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-3 px-4 text-right">Литры</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-3 px-4 text-right">Сумма, ₽</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-3 px-4 text-left">Статус</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-3 px-4 text-left">Чек</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-3 px-4 text-left">Ошибка</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="tx in filteredTransactions"
            :key="tx.id"
            class="border-b border-fuel-olive/10 last:border-b-0 hover:bg-fuel-cream/40"
          >
            <td class="font-karla text-sm text-fuel-forest py-3 px-4">{{ tx.id }}</td>
            <td class="font-karla text-sm text-fuel-olive py-3 px-4">{{ formatTimestamp(tx.createdAt) }}</td>
            <td class="font-rubik font-medium text-fuel-forest py-3 px-4">{{ tx.fuelType }}</td>
            <td class="font-karla text-sm text-fuel-forest py-3 px-4 text-right">{{ formatLiters(tx.liters) }}</td>
            <td class="font-rubik font-semibold text-fuel-forest py-3 px-4 text-right">
              {{ formatAmount(tx.amountRub) }}
            </td>
            <td class="py-3 px-4">
              <span
                class="inline-flex items-center rounded-full px-3 py-1 text-xs font-karla font-medium"
                :class="statusBadgeClass(tx.status)"
              >
                {{ statusLabel(tx.status) }}
              </span>
            </td>
            <td class="font-karla text-sm text-fuel-olive py-3 px-4">{{ tx.receiptNumber || '—' }}</td>
            <td class="font-karla text-sm text-red-600 py-3 px-4">{{ tx.errorMessage || '' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>
