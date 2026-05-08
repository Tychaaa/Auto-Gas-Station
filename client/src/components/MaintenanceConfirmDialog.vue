<script setup lang="ts">
import { ref } from 'vue'

defineProps<{
  fuelType?: string
}>()

const emit = defineEmits<{
  confirm: [reason: string]
  cancel: []
}>()

const reasonInput = ref('')
</script>

<template>
  <div
    class="fixed inset-0 z-[1000] flex items-center justify-center p-6 bg-fuel-forest/55 backdrop-blur-sm"
    role="dialog"
    aria-modal="true"
    aria-labelledby="maintenance-confirm-title"
    @click.self="emit('cancel')"
  >
    <div
      class="w-full max-w-sm rounded-2xl bg-fuel-cream shadow-2xl border border-fuel-olive/20 overflow-hidden"
    >
      <header class="bg-fuel-forest px-6 py-5 text-center">
        <h2
          id="maintenance-confirm-title"
          class="font-rubik font-bold text-2xl text-white leading-tight"
        >
          Киоск сейчас используется
        </h2>
      </header>

      <div class="px-6 py-7 flex flex-col gap-5">
        <div class="text-center space-y-2">
          <p class="font-karla text-base text-fuel-olive leading-snug">
            Пользователь выбирает объём заказа
            <template v-if="fuelType">
              ({{ fuelType }})
            </template>.
            Перевод в техобслуживание прервёт незавершённую транзакцию.
          </p>
        </div>

        <label class="flex flex-col gap-2">
          <span class="font-karla text-sm text-fuel-forest">
            Причина (необязательно — отобразится на экране киоска)
          </span>
          <input
            v-model="reasonInput"
            type="text"
            placeholder="Например: замена картриджа с чеками"
            class="rounded-lg border border-fuel-olive/40 bg-white px-4 py-3
                   font-karla text-base text-fuel-forest placeholder:text-fuel-olive/60
                   focus:outline-none focus:ring-2 focus:ring-fuel-lime focus:border-fuel-lime"
            maxlength="200"
          />
        </label>

        <div class="flex flex-col gap-3">
          <button
            type="button"
            class="w-full font-rubik font-semibold text-lg px-8 py-3.5 rounded-xl
                   bg-amber-500 text-white hover:bg-amber-600 active:scale-[0.98]
                   transition-all duration-200 shadow-md shadow-amber-400/25
                   focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-amber-500 focus-visible:ring-offset-2 focus-visible:ring-offset-fuel-cream"
            @click="emit('confirm', reasonInput)"
          >
            Перевести и прервать
          </button>
          <button
            type="button"
            class="w-full font-rubik font-semibold text-lg px-8 py-3.5 rounded-xl
                   border border-fuel-olive/30 text-fuel-olive bg-white
                   hover:border-fuel-olive/50 hover:bg-fuel-cream/60 active:scale-[0.98]
                   transition-all duration-200
                   focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime focus-visible:ring-offset-2 focus-visible:ring-offset-fuel-cream"
            @click="emit('cancel')"
          >
            Отмена
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
