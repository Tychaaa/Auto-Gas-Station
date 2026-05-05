import { onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { useTransactionFlowStore } from '@/stores/transactionFlow'

const INACTIVITY_MS = Number(import.meta.env.VITE_KIOSK_IDLE_TIMEOUT_MS) || 3 * 60 * 1000

// Только тап/клик и клавиша — pointermove намеренно не отслеживается,
// чтобы пассивный hover не сбрасывал таймер.
const ACTIVITY_EVENTS = ['pointerdown', 'keydown'] as const

export function useInactivityTimeout() {
  const router = useRouter()
  const route = useRoute()
  const flowStore = useTransactionFlowStore()
  let timerId: ReturnType<typeof setTimeout> | null = null

  function resetTimer() {
    if (timerId !== null) clearTimeout(timerId)
    timerId = setTimeout(onTimeout, INACTIVITY_MS)
  }

  async function onTimeout() {
    // Административные маршруты не подпадают под таймаут киоска
    if (route.path.startsWith('/admin')) {
      resetTimer()
      return
    }

    const canGoHome = await flowStore.handleInactivityTimeout()
    if (canGoHome) {
      await router.push('/select/fuel')
    } else {
      // Транзакция в небезопасном состоянии — сбросить таймер и ждать дальше
      resetTimer()
    }
  }

  const handleActivity = () => resetTimer()

  onMounted(() => {
    ACTIVITY_EVENTS.forEach((e) => window.addEventListener(e, handleActivity, { passive: true }))
    resetTimer()
  })

  onUnmounted(() => {
    ACTIVITY_EVENTS.forEach((e) => window.removeEventListener(e, handleActivity))
    if (timerId !== null) clearTimeout(timerId)
  })
}
