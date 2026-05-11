<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

import {
  getAdminTransaction,
  listAdminTransactions,
  type AdminTransactionDetailsView,
  type AdminTransactionView,
} from '@/api/admin.api'

const STATUS_LABELS: Record<string, string> = {
  selection: 'Выбор',
  payment_pending: 'Ожидание оплаты',
  paid: 'Оплачено',
  fueling: 'Отпуск',
  fiscalizing: 'Фискализация',
  completed: 'Завершено',
  failed: 'Ошибка',
  abandoned: 'Отменено',
}

const PAYMENT_STATUS_LABELS: Record<string, string> = {
  none: '—',
  pending: 'Ожидание',
  approved: 'Одобрено',
  declined: 'Отклонено',
}

const FISCAL_STATUS_LABELS: Record<string, string> = {
  none: '—',
  pending: 'Ожидание',
  done: 'Выдан',
  failed: 'Ошибка',
}

const FUELING_STATUS_LABELS: Record<string, string> = {
  none: '—',
  starting: 'Запуск',
  dispensing: 'Отпуск',
  completed_waiting_fiscal: 'Ожидание чека',
  failed: 'Ошибка',
}

const ORDER_MODE_LABELS: Record<string, string> = {
  amount: 'По сумме',
  liters: 'По литрам',
  preset: 'Пресет',
}

const STATUS_BADGE_CLASSES: Record<string, string> = {
  completed: 'bg-fuel-lime/20 text-fuel-forest',
  failed: 'bg-red-100 text-red-700',
  abandoned: 'bg-gray-100 text-gray-600',
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

const selectedTx = ref<AdminTransactionDetailsView | null>(null)
const isDetailsOpen = ref(false)
const isDetailsLoading = ref(false)
const detailsError = ref<string | null>(null)

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
  if (!iso) return '—'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
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

async function openDetails(id: string): Promise<void> {
  isDetailsOpen.value = true
  isDetailsLoading.value = true
  detailsError.value = null
  selectedTx.value = null
  try {
    selectedTx.value = await getAdminTransaction(id)
  } catch (error) {
    detailsError.value =
      error instanceof Error ? error.message : 'Не удалось загрузить детали транзакции.'
  } finally {
    isDetailsLoading.value = false
  }
}

function closeDetails(): void {
  isDetailsOpen.value = false
  selectedTx.value = null
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
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-3 px-4 text-left whitespace-nowrap">ID</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-3 px-4 text-left whitespace-nowrap">Время</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-3 px-4 text-left whitespace-nowrap">Топливо</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-3 px-4 text-right whitespace-nowrap">Литры</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-3 px-4 text-right whitespace-nowrap">Сумма, ₽</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-3 px-4 text-left whitespace-nowrap">Статус</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-3 px-4 text-left whitespace-nowrap">Чек</th>
            <th class="font-karla text-xs uppercase tracking-widest text-fuel-olive py-3 px-4 text-left whitespace-nowrap">Ошибка</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="tx in filteredTransactions"
            :key="tx.id"
            class="border-b border-fuel-olive/10 last:border-b-0 hover:bg-fuel-cream/40 cursor-pointer"
            @click="openDetails(tx.id)"
          >
            <td class="font-karla text-sm text-fuel-forest py-3 px-4 font-mono text-xs">{{ tx.id }}</td>
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
            <td class="font-karla text-sm text-red-600 py-3 px-4 max-w-xs truncate">{{ tx.errorMessage || '' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>

  <!-- Модалка деталей транзакции -->
  <Teleport to="body">
    <div
      v-if="isDetailsOpen"
      class="fixed inset-0 z-50 flex items-start justify-center bg-black/40 overflow-y-auto py-8"
      @click.self="closeDetails"
    >
      <div class="bg-white rounded-2xl shadow-xl w-full max-w-2xl mx-4 my-auto">
        <div class="flex items-center justify-between px-6 py-4 border-b border-fuel-olive/20">
          <h4 class="font-rubik font-semibold text-lg text-fuel-forest">
            Детали транзакции
          </h4>
          <button
            type="button"
            class="text-fuel-olive hover:text-fuel-forest transition-colors text-xl leading-none"
            @click="closeDetails"
          >
            ✕
          </button>
        </div>

        <div class="px-6 py-5">
          <p v-if="isDetailsLoading" class="font-karla text-sm text-fuel-olive">
            Загружаем детали...
          </p>
          <p v-else-if="detailsError" class="font-karla text-sm text-red-600">
            {{ detailsError }}
          </p>
          <template v-else-if="selectedTx">
            <!-- Идентификатор и время -->
            <dl class="grid grid-cols-2 gap-x-6 gap-y-3 mb-6">
              <div class="col-span-2">
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">ID</dt>
                <dd class="font-mono text-sm text-fuel-forest break-all">{{ selectedTx.id }}</dd>
              </div>
              <div>
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Создана</dt>
                <dd class="font-karla text-sm text-fuel-forest">{{ formatTimestamp(selectedTx.createdAt) }}</dd>
              </div>
              <div>
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Обновлена</dt>
                <dd class="font-karla text-sm text-fuel-forest">{{ formatTimestamp(selectedTx.updatedAt) }}</dd>
              </div>
            </dl>

            <!-- Заказ -->
            <h5 class="font-rubik font-semibold text-sm text-fuel-forest mb-3 pb-1 border-b border-fuel-olive/15">Заказ</h5>
            <dl class="grid grid-cols-2 gap-x-6 gap-y-3 mb-6">
              <div>
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Топливо</dt>
                <dd class="font-rubik font-medium text-fuel-forest">{{ selectedTx.fuelType }}</dd>
              </div>
              <div>
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Способ</dt>
                <dd class="font-karla text-sm text-fuel-forest">{{ ORDER_MODE_LABELS[selectedTx.orderMode] ?? selectedTx.orderMode }}</dd>
              </div>
              <div v-if="selectedTx.amountRub > 0">
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Сумма заказа, ₽</dt>
                <dd class="font-karla text-sm text-fuel-forest">{{ selectedTx.amountRub }}</dd>
              </div>
              <div v-if="selectedTx.liters > 0">
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Литры заказа</dt>
                <dd class="font-karla text-sm text-fuel-forest">{{ selectedTx.liters.toFixed(2) }}</dd>
              </div>
              <div v-if="selectedTx.preset">
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Пресет</dt>
                <dd class="font-karla text-sm text-fuel-forest">{{ selectedTx.preset }}</dd>
              </div>
            </dl>

            <!-- Snapshot цены -->
            <h5 class="font-rubik font-semibold text-sm text-fuel-forest mb-3 pb-1 border-b border-fuel-olive/15">Снимок цены</h5>
            <dl class="grid grid-cols-2 gap-x-6 gap-y-3 mb-6">
              <div>
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Версия цен</dt>
                <dd class="font-karla text-sm text-fuel-forest">{{ selectedTx.priceVersionTag || '—' }}</dd>
              </div>
              <div>
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Валюта</dt>
                <dd class="font-karla text-sm text-fuel-forest">{{ selectedTx.currency || '—' }}</dd>
              </div>
              <div>
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Цена за литр, ₽</dt>
                <dd class="font-karla text-sm text-fuel-forest">{{ selectedTx.unitPriceRub.toFixed(2) }}</dd>
              </div>
              <div>
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Итоговая сумма, ₽</dt>
                <dd class="font-rubik font-semibold text-fuel-forest">{{ selectedTx.computedAmountRub.toFixed(2) }}</dd>
              </div>
              <div v-if="selectedTx.pricingSnapshotAt">
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Снимок зафиксирован</dt>
                <dd class="font-karla text-sm text-fuel-forest">{{ formatTimestamp(selectedTx.pricingSnapshotAt) }}</dd>
              </div>
              <div v-if="selectedTx.priceLockedUntil">
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Цена заблокирована до</dt>
                <dd class="font-karla text-sm text-fuel-forest">{{ formatTimestamp(selectedTx.priceLockedUntil) }}</dd>
              </div>
              <div v-if="selectedTx.priceWasRepriced">
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Цена пересчитана</dt>
                <dd class="font-karla text-sm text-amber-700">Да</dd>
              </div>
            </dl>

            <!-- Статусы -->
            <h5 class="font-rubik font-semibold text-sm text-fuel-forest mb-3 pb-1 border-b border-fuel-olive/15">Статусы</h5>
            <dl class="grid grid-cols-2 gap-x-6 gap-y-3 mb-6">
              <div>
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Общий</dt>
                <dd>
                  <span
                    class="inline-flex items-center rounded-full px-3 py-1 text-xs font-karla font-medium"
                    :class="statusBadgeClass(selectedTx.status)"
                  >
                    {{ statusLabel(selectedTx.status) }}
                  </span>
                </dd>
              </div>
              <div>
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Оплата</dt>
                <dd class="font-karla text-sm text-fuel-forest">{{ PAYMENT_STATUS_LABELS[selectedTx.paymentStatus] ?? selectedTx.paymentStatus }}</dd>
              </div>
              <div>
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Фискализация</dt>
                <dd class="font-karla text-sm text-fuel-forest">{{ FISCAL_STATUS_LABELS[selectedTx.fiscalStatus] ?? selectedTx.fiscalStatus }}</dd>
              </div>
              <div>
                <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Отпуск топлива</dt>
                <dd class="font-karla text-sm text-fuel-forest">{{ FUELING_STATUS_LABELS[selectedTx.fuelingStatus] ?? selectedTx.fuelingStatus }}</dd>
              </div>
            </dl>

            <!-- Оплата -->
            <template v-if="selectedTx.paymentProvider || selectedTx.paymentError">
              <h5 class="font-rubik font-semibold text-sm text-fuel-forest mb-3 pb-1 border-b border-fuel-olive/15">Оплата</h5>
              <dl class="grid grid-cols-2 gap-x-6 gap-y-3 mb-6">
                <div v-if="selectedTx.paymentProvider">
                  <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Провайдер</dt>
                  <dd class="font-karla text-sm text-fuel-forest">{{ selectedTx.paymentProvider }}</dd>
                </div>
                <div v-if="selectedTx.paymentSessionId">
                  <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Сессия</dt>
                  <dd class="font-mono text-xs text-fuel-forest break-all">{{ selectedTx.paymentSessionId }}</dd>
                </div>
                <div v-if="selectedTx.paymentError" class="col-span-2">
                  <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Ошибка оплаты</dt>
                  <dd class="font-karla text-sm text-red-600">{{ selectedTx.paymentError }}</dd>
                </div>
              </dl>
            </template>

            <!-- Фискализация -->
            <template v-if="selectedTx.receiptNumber || selectedTx.fiscalError">
              <h5 class="font-rubik font-semibold text-sm text-fuel-forest mb-3 pb-1 border-b border-fuel-olive/15">Фискализация</h5>
              <dl class="grid grid-cols-1 gap-y-3 mb-6">
                <div v-if="selectedTx.receiptNumber">
                  <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Номер чека</dt>
                  <dd class="font-karla text-sm text-fuel-forest">{{ selectedTx.receiptNumber }}</dd>
                </div>
                <div v-if="selectedTx.fiscalError">
                  <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Ошибка ККТ</dt>
                  <dd class="font-karla text-sm text-red-600">{{ selectedTx.fiscalError }}</dd>
                </div>
              </dl>
            </template>

            <!-- Налив -->
            <template v-if="selectedTx.fuelingSessionId || selectedTx.dispensedLiters > 0 || selectedTx.fuelingError">
              <h5 class="font-rubik font-semibold text-sm text-fuel-forest mb-3 pb-1 border-b border-fuel-olive/15">Отпуск топлива</h5>
              <dl class="grid grid-cols-2 gap-x-6 gap-y-3 mb-6">
                <div v-if="selectedTx.fuelingSessionId" class="col-span-2">
                  <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Сессия</dt>
                  <dd class="font-mono text-xs text-fuel-forest break-all">{{ selectedTx.fuelingSessionId }}</dd>
                </div>
                <div v-if="selectedTx.dispensedLiters > 0">
                  <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Отпущено, л</dt>
                  <dd class="font-rubik font-semibold text-fuel-forest">{{ selectedTx.dispensedLiters.toFixed(3) }}</dd>
                </div>
                <div v-if="selectedTx.dispenseComplete">
                  <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Завершён</dt>
                  <dd class="font-karla text-sm text-fuel-forest">
                    {{ selectedTx.dispensePartial ? 'Частично' : 'Полностью' }}
                  </dd>
                </div>
                <div v-if="selectedTx.fuelingError" class="col-span-2">
                  <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Ошибка налива</dt>
                  <dd class="font-karla text-sm text-red-600">{{ selectedTx.fuelingError }}</dd>
                </div>
              </dl>
            </template>

            <!-- Прочее -->
            <template v-if="selectedTx.abandonReason">
              <h5 class="font-rubik font-semibold text-sm text-fuel-forest mb-3 pb-1 border-b border-fuel-olive/15">Прочее</h5>
              <dl class="grid grid-cols-1 gap-y-3">
                <div>
                  <dt class="font-karla text-xs uppercase tracking-widest text-fuel-olive mb-1">Причина отмены</dt>
                  <dd class="font-karla text-sm text-fuel-forest">{{ selectedTx.abandonReason }}</dd>
                </div>
              </dl>
            </template>
          </template>
        </div>
      </div>
    </div>
  </Teleport>
</template>
