<script setup lang="ts">
defineProps<{
  /** Подписи шагов прогресса */
  steps: readonly string[]
  /** Номер текущего шага начиная с 1 */
  current: number
}>()
</script>

<template>
  <nav aria-label="Прогресс заправки" class="bg-white border-b border-zinc-200 shrink-0">
    <ol class="flex items-center justify-center px-10 py-4">
      <li
        v-for="(step, index) in steps"
        :key="step"
        class="flex items-center"
      >
        <!-- Круг с номером и подпись шага -->
        <div class="flex items-center gap-2.5">
          <div
            class="flex items-center justify-center w-8 h-8 rounded-full text-sm font-semibold font-rubik leading-none select-none transition-colors duration-300"
            :class="
              index + 1 <= current
                ? 'bg-zinc-900 text-white'
                : 'bg-white border-2 border-zinc-300 text-zinc-400'
            "
            :aria-current="index + 1 === current ? 'step' : undefined"
          >
            {{ index + 1 }}
          </div>

          <span
            class="text-sm font-karla transition-colors duration-300"
            :class="
              index + 1 === current
                ? 'text-zinc-900 font-semibold'
                : index + 1 < current
                  ? 'text-zinc-500 font-medium'
                  : 'text-zinc-400'
            "
          >
            {{ step }}
          </span>
        </div>

        <!-- Линия между шагами кроме последнего -->
        <div
          v-if="index < steps.length - 1"
          class="w-14 h-px mx-4 transition-colors duration-300"
          :class="index + 1 < current ? 'bg-zinc-500' : 'bg-zinc-300'"
          aria-hidden="true"
        />
      </li>
    </ol>
  </nav>
</template>
