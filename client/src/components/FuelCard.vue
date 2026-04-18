<script setup lang="ts">
defineProps<{
  /** Название топлива на карточке */
  name: string
  /** Короткая пометка под названием */
  grade: string
  /** Флаг выбранной карточки */
  selected?: boolean
}>()

defineEmits<{
  /** Событие выбора карточки */
  select: []
}>()
</script>

<template>
  <button
    type="button"
    :aria-pressed="selected"
    :aria-label="`${name} — ${grade}`"
    @click="$emit('select')"
    class="flex flex-col items-center justify-center gap-5 w-full py-10 px-6
           rounded-2xl bg-white cursor-pointer select-none
           border-2 transition-all duration-200 ease-out
           focus-visible:outline-hidden focus-visible:ring-2 focus-visible:ring-zinc-900 focus-visible:ring-offset-2"
    :class="[
      selected
        ? 'border-zinc-900 shadow-lg -translate-y-0.5'
        : 'border-zinc-200 hover:border-zinc-400 hover:shadow-sm hover:-translate-y-0.5',
    ]"
  >
    <!-- Иконка топлива -->
    <div
      class="flex items-center justify-center w-16 h-16 rounded-full transition-colors duration-200"
      :class="selected ? 'bg-zinc-900' : 'bg-zinc-100'"
    >
      <svg
        class="w-7 h-7 rotate-180 transition-colors duration-200"
        :class="selected ? 'text-white' : 'text-zinc-400'"
        viewBox="0 0 24 24"
        fill="currentColor"
        aria-hidden="true"
      >
        <path
          d="M12 2C8.13 2 5 5.13 5 9c0 5.25 7 13 7 13s7-7.75 7-13c0-3.87-3.13-7-7-7z"
        />
      </svg>
    </div>

    <!-- Название топлива -->
    <span class="font-rubik font-bold text-5xl leading-none text-zinc-900 tracking-tight">
      {{ name }}
    </span>

    <!-- Метка типа топлива -->
    <span
      class="font-karla text-xs font-semibold tracking-widest uppercase px-4 py-1.5 rounded-full transition-all duration-200"
      :class="selected ? 'bg-zinc-900 text-white' : 'bg-zinc-100 text-zinc-500'"
    >
      {{ grade }}
    </span>

    <!-- Индикатор выбора -->
    <div
      class="flex items-center justify-center w-6 h-6 rounded-full border-2 transition-all duration-200"
      :class="selected ? 'bg-zinc-900 border-zinc-900' : 'border-zinc-300'"
    >
      <svg
        v-if="selected"
        class="w-3.5 h-3.5 text-white"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        stroke-width="3"
        stroke-linecap="round"
        stroke-linejoin="round"
        aria-hidden="true"
      >
        <polyline points="20 6 9 17 4 12" />
      </svg>
    </div>
  </button>
</template>
