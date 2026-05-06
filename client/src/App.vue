<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'

import InactivityWarningDialog from '@/components/InactivityWarningDialog.vue'
import MaintenanceView from '@/views/MaintenanceView.vue'
import { useInactivityTimeout } from '@/composables/useInactivityTimeout'
import { useKioskStateStore } from '@/stores/kioskState'

const route = useRoute()
const kioskStateStore = useKioskStateStore()

const { isWarningVisible, secondsRemaining, cancelTimeout, triggerGoHome } = useInactivityTimeout()

// Админские маршруты не подпадают под режим тех работ
// Админ сам управляет этим режимом кнопкой в панели
const isAdminRoute = computed(() => route.path.startsWith('/admin'))

// Оверлей тех работ показываем только на киоск-маршрутах
const shouldShowMaintenance = computed(
  () => !isAdminRoute.value && kioskStateStore.maintenance,
)

// Поллинг включаем только в киоск-сессии чтобы админка не ловила оверлей
watch(
  isAdminRoute,
  (adminRoute) => {
    if (adminRoute) {
      kioskStateStore.stopPolling()
    } else {
      kioskStateStore.startPolling()
    }
  },
  { immediate: false },
)

onMounted(() => {
  if (!isAdminRoute.value) {
    kioskStateStore.startPolling()
  }
})

onBeforeUnmount(() => {
  kioskStateStore.stopPolling()
})
</script>

<template>
  <RouterView />
  <InactivityWarningDialog
    v-if="isWarningVisible"
    :seconds-remaining="secondsRemaining"
    @cancel="cancelTimeout"
    @go-home="triggerGoHome"
  />
  <MaintenanceView v-if="shouldShowMaintenance" />
</template>

<style scoped></style>
