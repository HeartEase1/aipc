<template>
  <div class="table-page-layout" :class="{ 'mobile-mode': isMobile }">
    <!-- 固定区域：操作按钮 -->
    <div v-if="$slots.actions" class="layout-section-fixed layout-section-actions">
      <slot name="actions" />
    </div>

    <!-- 固定区域：搜索和过滤器 -->
    <div v-if="$slots.filters" class="layout-section-fixed layout-section-filters">
      <slot name="filters" />
    </div>

    <!-- 滚动区域：表格 -->
    <div class="layout-section-scrollable layout-section-table">
      <div class="card table-scroll-container">
        <slot name="table" />
      </div>
    </div>

    <!-- 固定区域：分页器 -->
    <div v-if="$slots.pagination" class="layout-section-fixed layout-section-pagination">
      <slot name="pagination" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const isMobile = ref(false)

const checkMobile = () => {
  isMobile.value = window.innerWidth < 1024
}

onMounted(() => {
  checkMobile()
  window.addEventListener('resize', checkMobile)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkMobile)
})
</script>

<style scoped>
/* 桌面端：Flexbox 布局 */
.table-page-layout {
  @apply flex flex-col gap-6;
  /* Legacy defaults remain exactly 100vh - 64px - 4rem. ModernAppShell
     supplies the same contract through tokens when its header/padding differ. */
  height: calc(100vh - var(--console-header-height, 64px) - var(--console-page-vertical-padding, 4rem));
  height: calc(100dvh - var(--console-header-height, 64px) - var(--console-page-vertical-padding, 4rem));
}

.layout-section-fixed {
  @apply flex-shrink-0;
}

.layout-section-scrollable {
  @apply flex-1 min-h-0 flex flex-col;
}

/* 表格滚动容器 - 增强版表体滚动方案 */
.table-scroll-container {
  @apply flex flex-col overflow-hidden h-full bg-white dark:bg-dark-800 rounded-2xl border border-gray-200 dark:border-dark-700 shadow-sm;
}

.table-scroll-container :deep(.table-wrapper) {
  @apply flex-1 overflow-x-auto overflow-y-auto;
  /* 确保横向滚动条显示在最底部 */
  scrollbar-gutter: stable;
}

.table-scroll-container :deep(table) {
  @apply w-full;
  min-width: max-content; /* 关键：确保表格宽度根据内容撑开，从而触发横向滚动 */
  display: table; /* 使用标准 table 布局以支持 sticky 列 */
}

.table-scroll-container :deep(thead) {
  @apply bg-gray-50/80 dark:bg-dark-800/80 backdrop-blur-sm;
}

.table-scroll-container :deep(tbody) {
  /* 保持默认 table-row-group 显示，不使用 block */
}

.table-scroll-container :deep(th) {
  @apply px-5 py-4 text-left text-sm font-medium text-gray-600 dark:text-dark-300 border-b border-gray-200 dark:border-dark-700;
}

.table-scroll-container :deep(td) {
  @apply px-5 py-4 text-sm text-gray-700 dark:text-gray-300 border-b border-gray-100 dark:border-dark-800;
}

/* 移动端：恢复正常滚动 */
.table-page-layout.mobile-mode .table-scroll-container {
  @apply h-auto overflow-visible border-none shadow-none bg-transparent;
}

.table-page-layout.mobile-mode .layout-section-scrollable {
  @apply flex-none min-h-fit;
}

.table-page-layout.mobile-mode .table-scroll-container :deep(table) {
  @apply flex-none;
  display: table;
  min-width: 100%;
}

</style>

<style>
/* The modern variant stays in this component's lazy CSS chunk but is not
   scoped, so Vue cannot rewrite the ancestor selector. The shell prefix keeps
   the classic console and public pages unchanged. */
.modern-console-shell .table-page-layout {
  display: grid;
  grid-template-areas:
    'actions'
    'filters'
    'table'
    'pagination';
  grid-template-rows: auto auto minmax(0, 1fr) auto;
  gap: 0;
  height: var(--console-viewport-available-height);
  min-height: 0;
}

.modern-console-shell .table-page-layout > * {
  min-width: 0;
}

.modern-console-shell .table-page-layout > * + * {
  margin-block-start: 1.5rem;
}

.modern-console-shell .layout-section-actions {
  grid-area: actions;
}

.modern-console-shell .layout-section-filters {
  grid-area: filters;
}

.modern-console-shell .layout-section-table {
  grid-area: table;
}

.modern-console-shell .layout-section-pagination {
  grid-area: pagination;
}

@media (max-width: 1023px) {
  .modern-console-shell .table-page-layout.mobile-mode {
    grid-template-rows: auto auto auto auto;
    height: auto;
  }
}
</style>
