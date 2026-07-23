<template>
  <div class="flex flex-col items-center gap-5 sm:flex-row sm:gap-6">
    <div class="h-48 w-48 shrink-0">
      <Doughnut
        :data="data"
        :options="options"
        role="img"
        :aria-label="chartLabel"
      />
    </div>
    <div class="max-h-52 w-full min-w-0 flex-1 overflow-auto">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ArcElement, Chart as ChartJS, Legend, Tooltip, type ChartData, type ChartOptions } from 'chart.js'
import { Doughnut } from 'vue-chartjs'

defineProps<{
  data: ChartData<'doughnut'>
  options: ChartOptions<'doughnut'>
  chartLabel: string
}>()

ChartJS.register(ArcElement, Tooltip, Legend)
</script>
