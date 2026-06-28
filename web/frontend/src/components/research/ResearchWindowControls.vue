<template>
  <div class="research-window-controls">
    <NButtonGroup size="small">
      <NButton
        v-for="preset in timeRangePresets"
        :key="preset"
        secondary
        size="small"
        :aria-label="t(`research.timeRange.${preset}`)"
        :disabled="loading"
        :title="t(`research.timeRange.${preset}`)"
        @click="$emit('range', preset)"
      >
        {{ t(`research.timeRange.${preset}`) }}
      </NButton>
    </NButtonGroup>
    <NButton
      circle
      secondary
      size="small"
      :aria-label="t('research.previousWindow')"
      :disabled="!canLoadPrevious || loading"
      :title="t('research.previousWindow')"
      @click="$emit('previous')"
    >
      <template #icon>
        <ChevronLeft :size="16" />
      </template>
    </NButton>
    <NButton
      circle
      secondary
      size="small"
      :aria-label="t('research.nextWindow')"
      :disabled="!canLoadNext || loading"
      :title="t('research.nextWindow')"
      @click="$emit('next')"
    >
      <template #icon>
        <ChevronRight :size="16" />
      </template>
    </NButton>
  </div>
</template>

<script setup lang="ts">
import { ChevronLeft, ChevronRight } from "@lucide/vue";
import { NButton, NButtonGroup } from "naive-ui";
import { useI18n } from "vue-i18n";

import type { ResearchTimeRangePreset } from "@/composables/researchWorkspaceHelpers";

defineProps<{
  canLoadNext: boolean;
  canLoadPrevious: boolean;
  loading: boolean;
}>();

defineEmits<{
  next: [];
  previous: [];
  range: [preset: ResearchTimeRangePreset];
}>();

const { t } = useI18n();
const timeRangePresets: ResearchTimeRangePreset[] = ["latest", "1h", "6h", "1d"];
</script>

<style scoped>
.research-window-controls { display: inline-flex; align-items: center; gap: 6px; }

.research-window-controls :deep(.n-button) { min-width: 34px; }
</style>
