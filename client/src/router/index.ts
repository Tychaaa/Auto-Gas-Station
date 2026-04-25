import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'

import { useTransactionFlowStore } from '@/stores'

// Дополнительные флаги доступа для маршрутов
declare module 'vue-router' {
  interface RouteMeta {
    requiresTransaction?: boolean
    requiresPaymentPending?: boolean
    requiresPaymentFinished?: boolean
    requiresFuelingDone?: boolean
    requiresValidSelectionDraft?: boolean
  }
}

// Пути шагов пользовательского сценария
const FLOW_PATHS = {
  fuelSelect: '/select/fuel',
  orderSelect: '/select/order',
  paymentMethod: '/payment/method',
  paymentPending: '/payment/pending',
  paymentResult: '/payment/result',
  fuelingProgress: '/fueling/progress',
  fuelingDone: '/fueling/done',
} as const

// Маршруты приложения для сценария заправки
const routes: RouteRecordRaw[] = [
  {
    path: '/',
    redirect: FLOW_PATHS.fuelSelect,
  },
  {
    path: FLOW_PATHS.fuelSelect,
    name: 'fuel-select',
    component: () => import('@/views/FuelSelectView.vue'),
  },
  {
    path: FLOW_PATHS.orderSelect,
    name: 'order-select',
    component: () => import('@/views/OrderParamsView.vue'),
  },
  {
    path: FLOW_PATHS.paymentMethod,
    name: 'payment-method',
    component: () => import('@/views/PaymentMethodView.vue'),
    meta: {
      requiresValidSelectionDraft: true,
    },
  },
  {
    path: FLOW_PATHS.paymentPending,
    name: 'payment-pending',
    component: () => import('@/views/PaymentPendingView.vue'),
    meta: {
      requiresTransaction: true,
      requiresPaymentPending: true,
    },
  },
  {
    path: FLOW_PATHS.paymentResult,
    name: 'payment-result',
    component: () => import('@/views/PaymentResultView.vue'),
    meta: {
      requiresTransaction: true,
      requiresPaymentFinished: true,
    },
  },
  {
    path: FLOW_PATHS.fuelingProgress,
    name: 'fueling-progress',
    component: () => import('@/views/FuelingProgressView.vue'),
  },
  {
    path: FLOW_PATHS.fuelingDone,
    name: 'fueling-done',
    component: () => import('@/views/FuelingDoneView.vue'),
    meta: {
      requiresTransaction: true,
      requiresFuelingDone: true,
    },
  },
  {
    path: '/:pathMatch(.*)*',
    redirect: FLOW_PATHS.fuelSelect,
  },
]

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes,
})

// Защищает шаги сценария от перехода в неверном состоянии
router.beforeEach((to) => {
  const transactionFlowStore = useTransactionFlowStore()
  const transactionStatus = transactionFlowStore.transaction?.status

  if (to.meta.requiresTransaction && !transactionFlowStore.hasActiveTransaction) {
    return { path: FLOW_PATHS.fuelSelect }
  }

  if (to.meta.requiresValidSelectionDraft && !transactionFlowStore.isSelectionDraftValid) {
    return { path: FLOW_PATHS.orderSelect }
  }

  if (to.meta.requiresPaymentPending && transactionStatus !== 'payment_pending') {
    if (transactionStatus === 'paid' || transactionStatus === 'failed') {
      return { path: FLOW_PATHS.paymentResult }
    }
    if (transactionFlowStore.isSelectionDraftValid) {
      return { path: FLOW_PATHS.paymentMethod }
    }
    return { path: FLOW_PATHS.orderSelect }
  }

  if (to.meta.requiresPaymentFinished && transactionStatus !== 'paid' && transactionStatus !== 'failed') {
    if (transactionStatus === 'payment_pending') {
      return { path: FLOW_PATHS.paymentPending }
    }
    if (transactionFlowStore.isSelectionDraftValid) {
      return { path: FLOW_PATHS.paymentMethod }
    }
    return { path: FLOW_PATHS.orderSelect }
  }

  if (to.meta.requiresFuelingDone) {
    const fuelingStatus = transactionFlowStore.transaction?.fuelingStatus
    const isFuelingDoneState =
      transactionStatus === 'completed' ||
      transactionStatus === 'fiscalizing' ||
      fuelingStatus === 'completed_waiting_fiscal'

    if (isFuelingDoneState) {
      return true
    }

    if (transactionStatus === 'fueling') {
      return { path: FLOW_PATHS.fuelingProgress }
    }
    return { path: FLOW_PATHS.fuelSelect }
  }

  return true
})

export default router
