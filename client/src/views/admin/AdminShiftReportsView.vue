<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'

import {
  captureCalcReport,
  closeShift,
  createHeaderLine,
  deleteCalcReport,
  deleteHeaderLine,
  deleteShiftReport,
  getShiftStatus,
  listCalcReports,
  listHeaderLines,
  listShiftReports,
  openShift,
  updateHeaderLine,
  type AdminCalcReport,
  type AdminHeaderLine,
  type AdminShiftReport,
  type AdminShiftStatus,
} from '@/api/admin.api'

// ─── Утилиты ─────────────────────────────────────────────────────────────────

function formatTs(iso: string | undefined): string {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString('ru-RU', { hour12: false })
}

function formatHours(h: number): string {
  if (h < 0) return `−${Math.abs(h).toFixed(1)} ч`
  return `${h.toFixed(1)} ч`
}

function errMsg(e: unknown): string {
  return e instanceof Error ? e.message : 'Неизвестная ошибка'
}

// ─── A. Состояние смены ──────────────────────────────────────────────────────

const shiftStatus = ref<AdminShiftStatus | null>(null)
const shiftStatusError = ref<string | null>(null)
const isLoadingStatus = ref(false)
const isSubmittingOpen = ref(false)
const isSubmittingClose = ref(false)
const shiftActionError = ref<string | null>(null)
const shiftActionSuccess = ref<string | null>(null)

async function loadShiftStatus(): Promise<void> {
  isLoadingStatus.value = true
  shiftStatusError.value = null
  try {
    shiftStatus.value = await getShiftStatus()
  } catch (e) {
    shiftStatusError.value = errMsg(e)
  } finally {
    isLoadingStatus.value = false
  }
}

async function handleOpenShift(): Promise<void> {
  isSubmittingOpen.value = true
  shiftActionError.value = null
  shiftActionSuccess.value = null
  try {
    const r = await openShift()
    shiftActionSuccess.value = `Смена №${r.shiftNumber} открыта`
    await Promise.all([loadShiftStatus(), loadShiftReports()])
  } catch (e) {
    shiftActionError.value = errMsg(e)
  } finally {
    isSubmittingOpen.value = false
  }
}

async function handleCloseShift(): Promise<void> {
  isSubmittingClose.value = true
  shiftActionError.value = null
  shiftActionSuccess.value = null
  try {
    const r = await closeShift()
    shiftActionSuccess.value = `Смена №${r.shiftNumber} закрыта (FD: ${r.fdNumber})`
    await Promise.all([loadShiftStatus(), loadShiftReports()])
  } catch (e) {
    shiftActionError.value = errMsg(e)
  } finally {
    isSubmittingClose.value = false
  }
}

let pollTimer: ReturnType<typeof setInterval> | null = null

function startPolling(): void {
  pollTimer = setInterval(() => {
    if (document.visibilityState === 'visible') {
      loadShiftStatus()
    }
  }, 30_000)
}

function stopPolling(): void {
  if (pollTimer !== null) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

// ─── B. Отчёт о состоянии расчётов (свежий снапшот) ─────────────────────────

const lastCalcSnapshot = ref<AdminCalcReport | null>(null)
const isCapturingCalc = ref(false)
const calcCaptureError = ref<string | null>(null)

async function handleCaptureCalc(): Promise<void> {
  isCapturingCalc.value = true
  calcCaptureError.value = null
  try {
    lastCalcSnapshot.value = await captureCalcReport()
    await loadCalcReports()
  } catch (e) {
    calcCaptureError.value = errMsg(e)
  } finally {
    isCapturingCalc.value = false
  }
}

// ─── C. История Z-отчётов ─────────────────────────────────────────────────────

const REPORTS_PAGE_SIZE = 5

const shiftReports = ref<AdminShiftReport[]>([])
const isLoadingShiftReports = ref(false)
const shiftReportsError = ref<string | null>(null)
const shiftReportsExpanded = ref(false)

async function loadShiftReports(): Promise<void> {
  isLoadingShiftReports.value = true
  shiftReportsError.value = null
  try {
    shiftReports.value = await listShiftReports()
  } catch (e) {
    shiftReportsError.value = errMsg(e)
  } finally {
    isLoadingShiftReports.value = false
  }
}

async function handleDeleteShiftReport(id: number): Promise<void> {
  if (!window.confirm('Удалить эту запись из локального журнала Z-отчётов?')) return
  try {
    await deleteShiftReport(id)
    await loadShiftReports()
  } catch (e) {
    window.alert('Не удалось удалить запись: ' + errMsg(e))
  }
}

// ─── D. История отчётов о состоянии расчётов ─────────────────────────────────

const calcReports = ref<AdminCalcReport[]>([])
const isLoadingCalcReports = ref(false)
const calcReportsError = ref<string | null>(null)
const calcReportsExpanded = ref(false)

async function loadCalcReports(): Promise<void> {
  isLoadingCalcReports.value = true
  calcReportsError.value = null
  try {
    calcReports.value = await listCalcReports()
  } catch (e) {
    calcReportsError.value = errMsg(e)
  } finally {
    isLoadingCalcReports.value = false
  }
}

async function handleDeleteCalcReport(id: number | undefined): Promise<void> {
  if (id == null) return
  if (!window.confirm('Удалить эту запись из локального журнала отчётов о расчётах?')) return
  try {
    await deleteCalcReport(id)
    await loadCalcReports()
  } catch (e) {
    window.alert('Не удалось удалить запись: ' + errMsg(e))
  }
}

// ─── E. Header lines ──────────────────────────────────────────────────────────

const headerLines = ref<AdminHeaderLine[]>([])
const isLoadingHeaderLines = ref(false)
const headerLinesError = ref<string | null>(null)

const newLinePosition = ref<string>('')
const newLineText = ref<string>('')
const isAddingLine = ref(false)
const addLineError = ref<string | null>(null)

const editingLineId = ref<number | null>(null)
const editPosition = ref<string>('')
const editText = ref<string>('')
const isSavingLine = ref(false)

async function loadHeaderLines(): Promise<void> {
  isLoadingHeaderLines.value = true
  headerLinesError.value = null
  try {
    headerLines.value = await listHeaderLines()
  } catch (e) {
    headerLinesError.value = errMsg(e)
  } finally {
    isLoadingHeaderLines.value = false
  }
}

async function handleAddLine(): Promise<void> {
  const position = parseInt(newLinePosition.value)
  if (!newLineText.value.trim()) {
    addLineError.value = 'Текст строки не может быть пустым'
    return
  }
  isAddingLine.value = true
  addLineError.value = null
  try {
    await createHeaderLine({
      position: Number.isNaN(position) ? 0 : position,
      text: newLineText.value.trim(),
    })
    newLinePosition.value = ''
    newLineText.value = ''
    await loadHeaderLines()
  } catch (e) {
    addLineError.value = errMsg(e)
  } finally {
    isAddingLine.value = false
  }
}

function startEditLine(line: AdminHeaderLine): void {
  editingLineId.value = line.id
  editPosition.value = String(line.position)
  editText.value = line.text
}

function cancelEditLine(): void {
  editingLineId.value = null
}

async function handleSaveLine(id: number): Promise<void> {
  const position = parseInt(editPosition.value)
  if (!editText.value.trim()) return
  isSavingLine.value = true
  try {
    await updateHeaderLine(id, {
      position: Number.isNaN(position) ? 0 : position,
      text: editText.value.trim(),
    })
    editingLineId.value = null
    await loadHeaderLines()
  } catch (e) {
    window.alert('Не удалось сохранить: ' + errMsg(e))
  } finally {
    isSavingLine.value = false
  }
}

async function handleDeleteLine(id: number): Promise<void> {
  if (!window.confirm('Удалить эту строку заголовка чека?')) return
  try {
    await deleteHeaderLine(id)
    await loadHeaderLines()
  } catch (e) {
    window.alert('Не удалось удалить: ' + errMsg(e))
  }
}

// ─── Lifecycle ────────────────────────────────────────────────────────────────

onMounted(async () => {
  await Promise.all([
    loadShiftStatus(),
    loadShiftReports(),
    loadCalcReports(),
    loadHeaderLines(),
  ])
  startPolling()
})

onBeforeUnmount(() => {
  stopPolling()
})
</script>

<template>
  <section class="flex flex-col gap-10">

    <!-- A. Состояние смены -->
    <div class="bg-white rounded-2xl border border-fuel-olive/20 p-6 shadow-sm">
      <div class="flex items-center justify-between gap-4 mb-5">
        <h3 class="font-rubik font-semibold text-xl text-fuel-forest">Состояние смены</h3>
        <button
          type="button"
          :disabled="isLoadingStatus"
          class="font-karla text-sm text-fuel-forest underline-offset-4 hover:underline disabled:opacity-50 cursor-pointer disabled:cursor-default"
          @click="loadShiftStatus"
        >
          Обновить
        </button>
      </div>

      <div v-if="shiftStatusError" class="text-red-600 font-karla text-sm mb-4">
        {{ shiftStatusError }}
      </div>

      <div v-if="shiftStatus" class="grid grid-cols-2 gap-4 mb-6 sm:grid-cols-3 lg:grid-cols-4">
        <div>
          <p class="font-karla text-xs text-fuel-olive uppercase tracking-wider mb-1">Статус</p>
          <span
            class="inline-block font-rubik font-medium text-sm px-3 py-1 rounded-full"
            :class="{
              'bg-green-100 text-green-800': shiftStatus.isOpen && !shiftStatus.isExpired,
              'bg-amber-100 text-amber-800': shiftStatus.isExpired,
              'bg-gray-100 text-gray-600': !shiftStatus.isOpen && !shiftStatus.isExpired,
            }"
          >
            {{ shiftStatus.isExpired ? 'Просрочена' : shiftStatus.isOpen ? 'Открыта' : 'Закрыта' }}
          </span>
        </div>

        <div>
          <p class="font-karla text-xs text-fuel-olive uppercase tracking-wider mb-1">Номер смены</p>
          <p class="font-rubik font-semibold text-fuel-forest">{{ shiftStatus.shiftNumber }}</p>
        </div>

        <div>
          <p class="font-karla text-xs text-fuel-olive uppercase tracking-wider mb-1">Открыта</p>
          <p class="font-karla text-fuel-forest text-sm">{{ shiftStatus.openedAt ? formatTs(shiftStatus.openedAt) : '—' }}</p>
        </div>

        <div>
          <p class="font-karla text-xs text-fuel-olive uppercase tracking-wider mb-1">Время работы</p>
          <p class="font-karla text-sm" :class="shiftStatus.hoursLeft < 0 ? 'text-amber-600 font-medium' : 'text-fuel-forest'">
            {{ shiftStatus.openedAt ? formatHours(shiftStatus.hoursOpen) : '—' }}
          </p>
        </div>

        <div>
          <p class="font-karla text-xs text-fuel-olive uppercase tracking-wider mb-1">Осталось</p>
          <p class="font-karla text-sm" :class="shiftStatus.hoursLeft < 0 ? 'text-red-600 font-medium' : 'text-fuel-forest'">
            {{ shiftStatus.openedAt ? formatHours(shiftStatus.hoursLeft) : '—' }}
          </p>
        </div>

        <div>
          <p class="font-karla text-xs text-fuel-olive uppercase tracking-wider mb-1">Чеков в смене</p>
          <p class="font-rubik font-semibold text-fuel-forest">{{ shiftStatus.receiptNum }}</p>
        </div>
      </div>

      <div v-if="shiftActionSuccess" class="text-green-700 font-karla text-sm mb-3">{{ shiftActionSuccess }}</div>
      <div v-if="shiftActionError" class="text-red-600 font-karla text-sm mb-3">{{ shiftActionError }}</div>

      <div class="flex flex-wrap gap-3">
        <button
          type="button"
          :disabled="isSubmittingOpen || (shiftStatus?.isOpen ?? false) || isLoadingStatus"
          class="font-rubik font-medium text-sm px-5 py-2.5 rounded-xl bg-fuel-forest text-white
                 hover:bg-fuel-forest/80 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
          @click="handleOpenShift"
        >
          {{ isSubmittingOpen ? 'Открываю…' : 'Открыть смену' }}
        </button>

        <button
          type="button"
          :disabled="isSubmittingClose || !(shiftStatus?.isOpen ?? true) || isLoadingStatus"
          class="font-rubik font-medium text-sm px-5 py-2.5 rounded-xl border border-fuel-forest text-fuel-forest
                 hover:bg-fuel-forest/10 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
          @click="handleCloseShift"
        >
          {{ isSubmittingClose ? 'Закрываю…' : 'Закрыть смену (Z-отчёт)' }}
        </button>
      </div>
    </div>

    <!-- B. Отчёт о состоянии расчётов -->
    <div class="bg-white rounded-2xl border border-fuel-olive/20 p-6 shadow-sm">
      <h3 class="font-rubik font-semibold text-xl text-fuel-forest mb-5">Отчёт о состоянии расчётов</h3>

      <div class="grid grid-cols-2 gap-4 bg-fuel-cream/40 rounded-xl p-4 sm:grid-cols-3 mb-5">
        <div>
          <p class="font-karla text-xs text-fuel-olive uppercase tracking-wider mb-1">FD-номер</p>
          <p class="font-rubik font-semibold text-fuel-forest">{{ lastCalcSnapshot?.fdNumber ?? '—' }}</p>
        </div>
        <div>
          <p class="font-karla text-xs text-fuel-olive uppercase tracking-wider mb-1">Фискальный признак</p>
          <p class="font-rubik font-semibold text-fuel-forest">{{ lastCalcSnapshot?.fiscalSign ?? '—' }}</p>
        </div>
        <div>
          <p class="font-karla text-xs text-fuel-olive uppercase tracking-wider mb-1">Неподтверждённых</p>
          <p class="font-rubik font-semibold text-fuel-forest">{{ lastCalcSnapshot?.unconfirmedCount ?? '—' }}</p>
        </div>
        <div>
          <p class="font-karla text-xs text-fuel-olive uppercase tracking-wider mb-1">Первый неподтверждённый</p>
          <p class="font-karla text-sm text-fuel-forest">{{ lastCalcSnapshot?.firstUnconfirmedDate ?? '—' }}</p>
        </div>
        <div>
          <p class="font-karla text-xs text-fuel-olive uppercase tracking-wider mb-1">Время ККТ</p>
          <p class="font-karla text-sm text-fuel-forest">{{ lastCalcSnapshot?.datetime ? formatTs(lastCalcSnapshot.datetime) : '—' }}</p>
        </div>
      </div>

      <div class="flex items-center gap-4">
        <button
          type="button"
          :disabled="isCapturingCalc"
          class="font-rubik font-medium text-sm px-5 py-2.5 rounded-xl bg-fuel-forest text-white
                 hover:bg-fuel-forest/80 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
          @click="handleCaptureCalc"
        >
          {{ isCapturingCalc ? 'Запрашиваю…' : 'Снять и сохранить' }}
        </button>
        <span v-if="calcCaptureError" class="font-karla text-sm text-red-600">{{ calcCaptureError }}</span>
      </div>
    </div>

    <!-- C. История Z-отчётов -->
    <div class="bg-white rounded-2xl border border-fuel-olive/20 p-6 shadow-sm">
      <div class="flex items-center justify-between gap-4 mb-5">
        <h3 class="font-rubik font-semibold text-xl text-fuel-forest">История Z-отчётов</h3>
        <button
          type="button"
          :disabled="isLoadingShiftReports"
          class="font-karla text-sm text-fuel-forest underline-offset-4 hover:underline disabled:opacity-50 cursor-pointer disabled:cursor-default"
          @click="loadShiftReports"
        >
          Обновить
        </button>
      </div>

      <div v-if="shiftReportsError" class="text-red-600 font-karla text-sm mb-3">{{ shiftReportsError }}</div>

      <div v-if="shiftReports.length === 0 && !isLoadingShiftReports" class="font-karla text-sm text-fuel-olive">
        Записей нет
      </div>

      <div v-else-if="shiftReports.length > 0" class="overflow-x-auto">
        <table class="w-full font-karla text-sm">
          <thead>
            <tr class="border-b border-fuel-olive/20 text-fuel-olive text-xs uppercase tracking-wider">
              <th class="text-left py-2 pr-4 font-medium">Закрыта</th>
              <th class="text-left py-2 pr-4 font-medium">Смена №</th>
              <th class="text-left py-2 pr-4 font-medium">FD-номер</th>
              <th class="text-left py-2 pr-4 font-medium">ФП</th>
              <th class="py-2"></th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="r in (shiftReportsExpanded ? shiftReports : shiftReports.slice(0, REPORTS_PAGE_SIZE))"
              :key="r.id"
              class="border-b border-fuel-olive/10 hover:bg-fuel-cream/20"
            >
              <td class="py-2 pr-4 text-fuel-forest">{{ formatTs(r.closedAt) }}</td>
              <td class="py-2 pr-4 text-fuel-forest font-medium">{{ r.shiftNumber }}</td>
              <td class="py-2 pr-4 text-fuel-forest">{{ r.fdNumber }}</td>
              <td class="py-2 pr-4 text-fuel-forest">{{ r.fiscalSign }}</td>
              <td class="py-2 text-right">
                <button
                  type="button"
                  class="font-karla text-xs text-red-500 hover:text-red-700 underline-offset-2 hover:underline cursor-pointer"
                  @click="handleDeleteShiftReport(r.id)"
                >
                  Удалить
                </button>
              </td>
            </tr>
          </tbody>
        </table>

        <div v-if="shiftReports.length > REPORTS_PAGE_SIZE" class="mt-3 flex justify-center">
          <button
            v-if="!shiftReportsExpanded"
            type="button"
            class="font-karla text-sm text-fuel-forest hover:underline cursor-pointer"
            @click="shiftReportsExpanded = true"
          >
            Показать больше ({{ shiftReports.length - REPORTS_PAGE_SIZE }})
          </button>
          <button
            v-else
            type="button"
            class="font-karla text-sm text-fuel-forest hover:underline cursor-pointer"
            @click="shiftReportsExpanded = false"
          >
            Скрыть
          </button>
        </div>
      </div>

      <p class="font-karla text-xs text-fuel-olive/70 mt-3">
        Удаление записи влияет только на локальный журнал; информация в фискальном накопителе сохраняется.
      </p>
    </div>

    <!-- D. История отчётов о состоянии расчётов -->
    <div class="bg-white rounded-2xl border border-fuel-olive/20 p-6 shadow-sm">
      <div class="flex items-center justify-between gap-4 mb-5">
        <h3 class="font-rubik font-semibold text-xl text-fuel-forest">История отчётов о состоянии расчётов</h3>
        <button
          type="button"
          :disabled="isLoadingCalcReports"
          class="font-karla text-sm text-fuel-forest underline-offset-4 hover:underline disabled:opacity-50 cursor-pointer disabled:cursor-default"
          @click="loadCalcReports"
        >
          Обновить
        </button>
      </div>

      <div v-if="calcReportsError" class="text-red-600 font-karla text-sm mb-3">{{ calcReportsError }}</div>

      <div v-if="calcReports.length === 0 && !isLoadingCalcReports" class="font-karla text-sm text-fuel-olive">
        Записей нет
      </div>

      <div v-else-if="calcReports.length > 0" class="overflow-x-auto">
        <table class="w-full font-karla text-sm">
          <thead>
            <tr class="border-b border-fuel-olive/20 text-fuel-olive text-xs uppercase tracking-wider">
              <th class="text-left py-2 pr-4 font-medium">Снят</th>
              <th class="text-left py-2 pr-4 font-medium">FD-номер</th>
              <th class="text-left py-2 pr-4 font-medium">ФП</th>
              <th class="text-left py-2 pr-4 font-medium">Неподтверж.</th>
              <th class="text-left py-2 pr-4 font-medium">Первый неподтв.</th>
              <th class="text-left py-2 pr-4 font-medium">Время ККТ</th>
              <th class="py-2"></th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="r in (calcReportsExpanded ? calcReports : calcReports.slice(0, REPORTS_PAGE_SIZE))"
              :key="r.id"
              class="border-b border-fuel-olive/10 hover:bg-fuel-cream/20"
            >
              <td class="py-2 pr-4 text-fuel-forest">{{ formatTs(r.createdAt) }}</td>
              <td class="py-2 pr-4 text-fuel-forest">{{ r.fdNumber }}</td>
              <td class="py-2 pr-4 text-fuel-forest">{{ r.fiscalSign }}</td>
              <td class="py-2 pr-4 text-fuel-forest font-medium">{{ r.unconfirmedCount }}</td>
              <td class="py-2 pr-4 text-fuel-forest">{{ r.firstUnconfirmedDate ?? '—' }}</td>
              <td class="py-2 pr-4 text-fuel-forest">{{ r.datetime ? formatTs(r.datetime) : '—' }}</td>
              <td class="py-2 text-right">
                <button
                  type="button"
                  class="font-karla text-xs text-red-500 hover:text-red-700 underline-offset-2 hover:underline cursor-pointer"
                  @click="handleDeleteCalcReport(r.id)"
                >
                  Удалить
                </button>
              </td>
            </tr>
          </tbody>
        </table>

        <div v-if="calcReports.length > REPORTS_PAGE_SIZE" class="mt-3 flex justify-center">
          <button
            v-if="!calcReportsExpanded"
            type="button"
            class="font-karla text-sm text-fuel-forest hover:underline cursor-pointer"
            @click="calcReportsExpanded = true"
          >
            Показать больше ({{ calcReports.length - REPORTS_PAGE_SIZE }})
          </button>
          <button
            v-else
            type="button"
            class="font-karla text-sm text-fuel-forest hover:underline cursor-pointer"
            @click="calcReportsExpanded = false"
          >
            Скрыть
          </button>
        </div>
      </div>

      <p class="font-karla text-xs text-fuel-olive/70 mt-3">
        Удаление записи влияет только на локальный журнал; информация в фискальном накопителе сохраняется.
      </p>
    </div>

    <!-- E. Строки-заголовки чека -->
    <div class="bg-white rounded-2xl border border-fuel-olive/20 p-6 shadow-sm">
      <div class="flex items-center justify-between gap-4 mb-5">
        <h3 class="font-rubik font-semibold text-xl text-fuel-forest">Строки-заголовки чека</h3>
        <button
          type="button"
          :disabled="isLoadingHeaderLines"
          class="font-karla text-sm text-fuel-forest underline-offset-4 hover:underline disabled:opacity-50 cursor-pointer disabled:cursor-default"
          @click="loadHeaderLines"
        >
          Обновить
        </button>
      </div>

      <div v-if="headerLinesError" class="text-red-600 font-karla text-sm mb-3">{{ headerLinesError }}</div>

      <div v-if="headerLines.length > 0" class="overflow-x-auto mb-6">
        <table class="w-full font-karla text-sm">
          <thead>
            <tr class="border-b border-fuel-olive/20 text-fuel-olive text-xs uppercase tracking-wider">
              <th class="text-left py-2 pr-4 font-medium w-20">Позиция</th>
              <th class="text-left py-2 pr-4 font-medium">Текст</th>
              <th class="py-2 w-32"></th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="line in headerLines"
              :key="line.id"
              class="border-b border-fuel-olive/10"
            >
              <template v-if="editingLineId === line.id">
                <td class="py-2 pr-4">
                  <input
                    v-model="editPosition"
                    type="number"
                    min="1"
                    class="w-16 border border-fuel-olive/40 rounded-lg px-2 py-1 text-sm font-karla focus:outline-none focus:ring-1 focus:ring-fuel-forest"
                  />
                </td>
                <td class="py-2 pr-4">
                  <input
                    v-model="editText"
                    type="text"
                    maxlength="40"
                    class="w-full border border-fuel-olive/40 rounded-lg px-2 py-1 text-sm font-karla focus:outline-none focus:ring-1 focus:ring-fuel-forest"
                  />
                </td>
                <td class="py-2 text-right">
                  <button
                    type="button"
                    :disabled="isSavingLine"
                    class="font-karla text-xs text-fuel-forest hover:underline mr-3 cursor-pointer disabled:opacity-50 disabled:cursor-default"
                    @click="handleSaveLine(line.id)"
                  >
                    Сохранить
                  </button>
                  <button
                    type="button"
                    class="font-karla text-xs text-fuel-olive hover:text-fuel-forest hover:underline cursor-pointer"
                    @click="cancelEditLine"
                  >
                    Отмена
                  </button>
                </td>
              </template>

              <template v-else>
                <td class="py-2 pr-4 text-fuel-forest font-medium">{{ line.position }}</td>
                <td class="py-2 pr-4 text-fuel-forest">{{ line.text }}</td>
                <td class="py-2 text-right">
                  <button
                    type="button"
                    class="font-karla text-xs text-fuel-forest hover:underline mr-3 cursor-pointer"
                    @click="startEditLine(line)"
                  >
                    Изменить
                  </button>
                  <button
                    type="button"
                    class="font-karla text-xs text-red-500 hover:text-red-700 hover:underline cursor-pointer"
                    @click="handleDeleteLine(line.id)"
                  >
                    Удалить
                  </button>
                </td>
              </template>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-else-if="!isLoadingHeaderLines" class="font-karla text-sm text-fuel-olive mb-6">
        Заголовочные строки не заданы
      </div>

      <!-- Форма добавления строки -->
      <div class="border-t border-fuel-olive/15 pt-5">
        <h4 class="font-rubik font-medium text-sm text-fuel-forest mb-3">Добавить строку</h4>
        <div class="flex flex-wrap items-end gap-3">
          <div>
            <label class="font-karla text-xs text-fuel-olive uppercase tracking-wider mb-1 block">Позиция</label>
            <input
              v-model="newLinePosition"
              type="number"
              min="1"
              placeholder="1"
              class="w-20 border border-fuel-olive/40 rounded-xl px-3 py-2 text-sm font-karla
                     focus:outline-none focus:ring-1 focus:ring-fuel-forest"
            />
          </div>
          <div class="flex-1 min-w-48">
            <label class="font-karla text-xs text-fuel-olive uppercase tracking-wider mb-1 block">Текст (макс. 40 символов)</label>
            <input
              v-model="newLineText"
              type="text"
              maxlength="40"
              placeholder=""
              class="w-full border border-fuel-olive/40 rounded-xl px-3 py-2 text-sm font-karla
                     focus:outline-none focus:ring-1 focus:ring-fuel-forest"
            />
          </div>
          <button
            type="button"
            :disabled="isAddingLine || !newLineText.trim()"
            class="font-rubik font-medium text-sm px-5 py-2 rounded-xl bg-fuel-forest text-white
                   hover:bg-fuel-forest/80 transition-colors cursor-pointer disabled:opacity-40 disabled:cursor-not-allowed"
            @click="handleAddLine"
          >
            {{ isAddingLine ? 'Добавляю…' : 'Добавить' }}
          </button>
        </div>
        <p v-if="addLineError" class="font-karla text-sm text-red-600 mt-2">{{ addLineError }}</p>
      </div>
    </div>

  </section>
</template>
