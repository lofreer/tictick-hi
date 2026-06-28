<template>
  <div class="research-window-controls">
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
import { NButton } from "naive-ui";
import { useI18n } from "vue-i18n";

defineProps<{
  canLoadNext: boolean;
  canLoadPrevious: boolean;
  loading: boolean;
}>();

defineEmits<{
  next: [];
  previous: [];
}>();

const { t } = useI18n();
</script>

<style scoped>
.research-window-controls { display: inline-flex; align-items: center; gap: 6px; }
</style>
