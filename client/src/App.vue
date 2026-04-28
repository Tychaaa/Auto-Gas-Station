<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
import { RouterView, useRoute } from 'vue-router'

import MaintenanceView from '@/views/MaintenanceView.vue'
import { useKioskStateStore } from '@/stores/kioskState'

const route = useRoute()
const kioskStateStore = useKioskStateStore()

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
  <MaintenanceView v-if="shouldShowMaintenance" />
</template>

<style scoped></style>
