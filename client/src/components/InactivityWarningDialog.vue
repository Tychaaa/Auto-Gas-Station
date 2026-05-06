<script setup lang="ts">
defineProps<{
  secondsRemaining: number
}>()

const emit = defineEmits<{
  cancel: []
  'go-home': []
}>()
</script>

<template>
  <!-- Затемнённый backdrop; тап по нему — то же, что «Отмена» -->
  <div
    class="fixed inset-0 z-[1000] flex items-center justify-center p-6 bg-fuel-forest/55 backdrop-blur-sm"
    role="dialog"
    aria-modal="true"
    aria-labelledby="inactivity-title"
    @click.self="emit('cancel')"
  >
    <!-- Карточка диалога -->
    <div
      class="w-full max-w-sm rounded-2xl bg-fuel-cream shadow-2xl border border-fuel-olive/20 overflow-hidden"
    >
      <!-- Шапка -->
      <header class="bg-fuel-forest px-6 py-5 text-center">
        <h2
          id="inactivity-title"
          class="font-rubik font-bold text-2xl text-white leading-tight"
        >
          Терминал не используется
        </h2>
      </header>

      <!-- Тело -->
      <div class="px-6 py-7 flex flex-col items-center gap-6">
        <!-- Круглый счётчик обратного отсчёта -->
        <div
          class="w-24 h-24 rounded-full bg-fuel-olive flex items-center justify-center shadow-lg shadow-fuel-lime/30"
          aria-hidden="true"
        >
          <span class="font-rubik font-bold text-4xl text-white leading-none">
            {{ secondsRemaining }}
          </span>
        </div>

        <!-- Поясняющий текст — статический, не скачет -->
        <div class="text-center space-y-2">
          <p class="font-rubik font-semibold text-xl text-fuel-forest">
            Никого нет рядом?
          </p>
          <p class="font-karla text-base text-fuel-olive leading-snug">
            Скоро произойдёт возврат на главный экран.
          </p>
        </div>

        <!-- Кнопки одинакового размера -->
        <div class="w-full flex flex-col gap-3">
          <button
            type="button"
            class="w-full font-rubik font-semibold text-lg px-8 py-3.5 rounded-xl bg-fuel-lime text-white hover:bg-fuel-forest active:scale-[0.98] transition-all duration-200 shadow-md shadow-fuel-lime/25 focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime focus-visible:ring-offset-2 focus-visible:ring-offset-fuel-cream"
            @click="emit('cancel')"
          >
            Отмена
          </button>
          <button
            type="button"
            class="w-full font-rubik font-semibold text-lg px-8 py-3.5 rounded-xl border border-fuel-olive/30 text-fuel-olive bg-white hover:border-fuel-olive/50 hover:bg-fuel-cream/60 active:scale-[0.98] transition-all duration-200 focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-fuel-lime focus-visible:ring-offset-2 focus-visible:ring-offset-fuel-cream"
            @click="emit('go-home')"
          >
            На главную
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
