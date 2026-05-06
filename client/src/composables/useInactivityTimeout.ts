import { onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { useTransactionFlowStore } from '@/stores/transactionFlow'

const INACTIVITY_MS = Number(import.meta.env.VITE_KIOSK_IDLE_TIMEOUT_MS) || 3 * 60 * 1000
const WARNING_MS = Number(import.meta.env.VITE_KIOSK_IDLE_WARNING_MS) || 60 * 1000
// Задержка до показа предупреждения; если WARNING_MS >= INACTIVITY_MS — предупреждение
// появляется сразу (нулевая задержка), таймаут по-прежнему срабатывает через WARNING_MS.
const WARNING_DELAY_MS = Math.max(0, INACTIVITY_MS - WARNING_MS)

// Только тап/клик и клавиша — pointermove намеренно не отслеживается,
// чтобы пассивный hover не сбрасывал таймер.
const ACTIVITY_EVENTS = ['pointerdown', 'keydown'] as const

export function useInactivityTimeout() {
  const router = useRouter()
  const route = useRoute()
  const flowStore = useTransactionFlowStore()

  const isWarningVisible = ref(false)
  const secondsRemaining = ref(0)

  let warningTimerId: ReturnType<typeof setTimeout> | null = null
  let timeoutTimerId: ReturnType<typeof setTimeout> | null = null
  let countdownTimerId: ReturnType<typeof setInterval> | null = null

  function clearAllTimers() {
    if (warningTimerId !== null) {
      clearTimeout(warningTimerId)
      warningTimerId = null
    }
    if (timeoutTimerId !== null) {
      clearTimeout(timeoutTimerId)
      timeoutTimerId = null
    }
    if (countdownTimerId !== null) {
      clearInterval(countdownTimerId)
      countdownTimerId = null
    }
  }

  function resetTimer() {
    clearAllTimers()
    isWarningVisible.value = false
    warningTimerId = setTimeout(showWarning, WARNING_DELAY_MS)
  }

  function isIdleScreen() {
    return route.path === '/select/fuel' || route.path === '/'
  }

  function showWarning() {
    if (route.path.startsWith('/admin') || isIdleScreen()) {
      resetTimer()
      return
    }

    isWarningVisible.value = true
    secondsRemaining.value = Math.round(WARNING_MS / 1000)

    countdownTimerId = setInterval(() => {
      if (secondsRemaining.value > 0) secondsRemaining.value--
    }, 1000)

    timeoutTimerId = setTimeout(onTimeout, WARNING_MS)
  }

  async function onTimeout() {
    clearAllTimers()
    isWarningVisible.value = false

    const canGoHome = await flowStore.handleInactivityTimeout()
    if (canGoHome) {
      await router.push('/select/fuel')
    } else {
      // Транзакция в небезопасном состоянии (оплата/налив) — сбросить таймер и ждать
      resetTimer()
    }
  }

  // Пользователь нажал «Отмена» — скрыть предупреждение и перезапустить таймер
  function cancelTimeout() {
    resetTimer()
  }

  // Пользователь нажал «На главную» — немедленный переход
  async function triggerGoHome() {
    clearAllTimers()
    isWarningVisible.value = false

    const canGoHome = await flowStore.handleInactivityTimeout()
    if (canGoHome) {
      await router.push('/select/fuel')
    } else {
      resetTimer()
    }
  }

  // Когда предупреждение видно — фоновые тапы игнорируем: пользователь должен
  // явно нажать одну из кнопок. Когда предупреждение скрыто — любое действие
  // сбрасывает таймер неактивности.
  const handleActivity = () => {
    if (!isWarningVisible.value) resetTimer()
  }

  // Если пользователь вернулся на главный экран (навигация назад, редирект и т.д.)
  // — скрыть предупреждение и остановить таймер.
  watch(
    () => route.path,
    (path) => {
      if (path === '/select/fuel' || path === '/') {
        clearAllTimers()
        isWarningVisible.value = false
      }
    },
  )

  onMounted(() => {
    ACTIVITY_EVENTS.forEach((e) => window.addEventListener(e, handleActivity, { passive: true }))
    resetTimer()
  })

  onUnmounted(() => {
    ACTIVITY_EVENTS.forEach((e) => window.removeEventListener(e, handleActivity))
    clearAllTimers()
  })

  return { isWarningVisible, secondsRemaining, cancelTimeout, triggerGoHome }
}
