import { watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

import { useKioskStateStore } from '@/stores/kioskState'
import { useTransactionFlowStore } from '@/stores/transactionFlow'

export function useMaintenanceAutoReset(): void {
  const route = useRoute()
  const kioskStateStore = useKioskStateStore()
  const transactionFlowStore = useTransactionFlowStore()
  const router = useRouter()

  watch(
    () => kioskStateStore.maintenance,
    (isNowMaintenance, wasMaintenance) => {
      if (wasMaintenance && !isNowMaintenance && !route.path.startsWith('/admin')) {
        transactionFlowStore.resetFlow()
        router.push('/select/fuel')
      }
    },
  )
}
