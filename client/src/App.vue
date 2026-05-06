<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import { RouterView, useRoute } from 'vue-router'

import MaintenanceView from '@/views/MaintenanceView.vue'
import { useKioskStateStore } from '@/stores/kioskState'

const route = useRoute()
const kioskStateStore = useKioskStateStore()

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
  <MaintenanceView v-if="shouldShowMaintenance" />
</template>

<style scoped></style>
