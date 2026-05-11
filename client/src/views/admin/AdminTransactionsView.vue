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

const ORDER_MODE_LABELS: Record<string, string> = {
  amount: 'по сумме',
  liters: 'по литрам',
  preset: 'пресет',
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

// ─── Подписи и стили событий ─────────────────────────────────────────────────

const EVENT_LABELS: Record<string, string> = {
  created: 'Транзакция создана',
  selection_updated: 'Выбор изменён',
  payment_started: 'Запуск оплаты',
  payment_approved: 'Оплата прошла',
  payment_declined: 'Оплата отклонена',
  fiscalizing_started: 'Фискализация',
  receipt_issued: 'Чек выдан',
  fiscal_failed: 'Ошибка фискализации',
  fueling_started: 'Налив запущен',
  fueling_dispensing: 'Идёт отпуск топлива',
  fueling_completed: 'Отпуск завершён',
  fueling_failed: 'Ошибка налива',
  completed: 'Операция завершена',
  failed: 'Ошибка',
  abandoned: 'Отменено',
}

const EVENT_DOT_CLASSES: Record<string, string> = {
  created: 'bg-fuel-olive',
  selection_updated: 'bg-amber-400',
  payment_started: 'bg-sky-400',
  payment_approved: 'bg-emerald-500',
  payment_declined: 'bg-red-500',
  fiscalizing_started: 'bg-sky-400',
  receipt_issued: 'bg-emerald-400',
  fiscal_failed: 'bg-red-500',
  fueling_started: 'bg-sky-500',
  fueling_dispensing: 'bg-sky-400',
  fueling_completed: 'bg-emerald-400',
  fueling_failed: 'bg-red-500',
  completed: 'bg-emerald-600',
  failed: 'bg-red-600',
  abandoned: 'bg-gray-400',
}

const EVENT_LABEL_CLASSES: Record<string, string> = {
  payment_declined: 'text-red-700',
  fiscal_failed: 'text-red-700',
  fueling_failed: 'text-red-700',
  failed: 'text-red-700',
  completed: 'text-emerald-700 font-semibold',
  abandoned: 'text-gray-500',
}

// ─── Данные ──────────────────────────────────────────────────────────────────

const transactions = ref<AdminTransactionView[]>([])
const isLoading = ref(false)
const loadError = ref<string | null>(null)

const statusFilter = ref<string>('all')

const selectedTx = ref<AdminTransactionDetailsView | null>(null)
const isDetailsOpen = ref(false)
const isDetailsLoading = ref(false)
const detailsError = ref<string | null>(null)

// ─── Вычисляемые ─────────────────────────────────────────────────────────────

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

// ─── Форматирование ───────────────────────────────────────────────────────────

function formatTimestamp(iso: string): string {
  if (!iso) return '—'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleString('ru-RU', { hour12: false })
}

function formatEventTime(iso: string): string {
  if (!iso) return '—'
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return iso
  return date.toLocaleTimeString('ru-RU', { hour12: false })
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

function eventLabel(eventType: string): string {
  return EVENT_LABELS[eventType] ?? eventType
}

function eventDotClass(eventType: string): string {
  return EVENT_DOT_CLASSES[eventType] ?? 'bg-fuel-olive/50'
}

function eventLabelClass(eventType: string): string {
  return EVENT_LABEL_CLASSES[eventType] ?? 'text-fuel-forest'
}

function orderSummary(tx: AdminTransactionDetailsView): string {
  const mode = ORDER_MODE_LABELS[tx.orderMode] ?? tx.orderMode
  if (tx.orderMode === 'amount') {
    return `${tx.fuelType} · ${tx.amountRub} ₽ ${mode}`
  }
  if (tx.orderMode === 'liters') {
    return `${tx.fuelType} · ${tx.liters.toFixed(2)} л ${mode}`
  }
  if (tx.orderMode === 'preset') {
    return `${tx.fuelType} · ${tx.preset} (${mode})`
  }
  return tx.fuelType
}

// ─── Действия ────────────────────────────────────────────────────────────────

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
      <div class="bg-white rounded-2xl shadow-xl w-full max-w-lg mx-4 my-auto">

        <!-- Шапка -->
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
          <!-- Загрузка / ошибка -->
          <p v-if="isDetailsLoading" class="font-karla text-sm text-fuel-olive">
            Загружаем детали...
          </p>
          <p v-else-if="detailsError" class="font-karla text-sm text-red-600">
            {{ detailsError }}
          </p>

          <template v-else-if="selectedTx">
            <!-- Краткое резюме транзакции -->
            <div class="bg-fuel-cream/50 rounded-xl p-4 mb-6 flex flex-col gap-2">
              <div class="flex items-start justify-between gap-3">
                <span class="font-mono text-xs text-fuel-olive break-all leading-relaxed">{{ selectedTx.id }}</span>
                <span
                  class="inline-flex items-center rounded-full px-3 py-1 text-xs font-karla font-medium whitespace-nowrap flex-shrink-0"
                  :class="statusBadgeClass(selectedTx.status)"
                >
                  {{ statusLabel(selectedTx.status) }}
                </span>
              </div>
              <div class="flex items-baseline justify-between gap-3">
                <span class="font-karla text-sm text-fuel-forest">
                  {{ orderSummary(selectedTx) }}
                </span>
                <span class="font-rubik font-semibold text-fuel-forest whitespace-nowrap">
                  {{ selectedTx.computedAmountRub > 0 ? selectedTx.computedAmountRub.toFixed(2) + ' ₽' : '' }}
                </span>
              </div>
              <span class="font-karla text-xs text-fuel-olive/70">
                {{ formatTimestamp(selectedTx.createdAt) }}
              </span>
            </div>

            <!-- Timeline событий -->
            <div v-if="selectedTx.events && selectedTx.events.length > 0">
              <h5 class="font-rubik font-semibold text-sm text-fuel-forest mb-4">
                История
              </h5>
              <div>
                <div
                  v-for="(event, index) in selectedTx.events"
                  :key="index"
                  class="flex gap-3"
                  :class="index < selectedTx.events.length - 1 ? 'pb-6' : ''"
                >
                  <!-- Колонка: кружок + линия точно по центру -->
                  <div class="relative flex-shrink-0 w-5 flex justify-center">
                    <!-- Линия внутри колонки — left-1/2 гарантирует центровку,
                         -bottom-6 тянет её через межстрочный pb-6 к следующему кружку -->
                    <div
                      v-if="index < selectedTx.events.length - 1"
                      class="absolute top-4 -bottom-6 w-px bg-fuel-olive/25 left-1/2 -translate-x-1/2"
                    />
                    <!-- Кружок поверх линии -->
                    <div
                      class="relative z-10 w-3 h-3 rounded-full mt-1 ring-2 ring-white shadow-sm"
                      :class="eventDotClass(event.eventType)"
                    />
                  </div>
                  <!-- Содержимое события -->
                  <div class="min-w-0 flex-1 pb-0.5">
                    <div class="flex items-baseline flex-wrap gap-x-2 gap-y-0.5">
                      <span
                        class="font-karla text-sm font-medium"
                        :class="eventLabelClass(event.eventType)"
                      >
                        {{ eventLabel(event.eventType) }}
                      </span>
                      <span
                        class="font-karla text-xs text-fuel-olive/60"
                        :title="formatTimestamp(event.occurredAt)"
                      >
                        {{ formatEventTime(event.occurredAt) }}
                      </span>
                    </div>
                    <p v-if="event.detail" class="font-karla text-xs text-fuel-olive mt-0.5 leading-relaxed">
                      {{ event.detail }}
                    </p>
                  </div>
                </div>
              </div>
            </div>

            <!-- Для старых транзакций без журнала -->
            <div v-else class="text-center py-4">
              <p class="font-karla text-sm text-fuel-olive/60">
                История событий для этой транзакции недоступна
              </p>
            </div>
          </template>
        </div>

      </div>
    </div>
  </Teleport>
</template>
