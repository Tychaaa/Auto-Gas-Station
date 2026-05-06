<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
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

const shouldShowMaintenance = computed(
  () => !isAdminRoute.value && kioskStateStore.maintenance,
)

onMounted(() => {
  kioskStateStore.connect()
})

onBeforeUnmount(() => {
  kioskStateStore.disconnect()
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
