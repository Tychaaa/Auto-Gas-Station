import { watch } from 'vue'
import { useRouter } from 'vue-router'

import { useKioskStateStore } from '@/stores/kioskState'
import { useTransactionFlowStore } from '@/stores/transactionFlow'

export function useMaintenanceAutoReset(): void {
  const kioskStateStore = useKioskStateStore()
  const transactionFlowStore = useTransactionFlowStore()
  const router = useRouter()

  watch(
    () => kioskStateStore.maintenance,
    (isNowMaintenance, wasMaintenance) => {
      if (wasMaintenance && !isNowMaintenance) {
        transactionFlowStore.resetFlow()
        router.push('/select/fuel')
      }
    },
  )
}
