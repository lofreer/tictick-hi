<template>
  <section class="page">
    <header class="page-header">
      <div>
        <h1 class="page-title">{{ t(titleKey) }}</h1>
        <p v-if="subtitleKey" class="page-subtitle">{{ t(subtitleKey) }}</p>
      </div>
    </header>
    <div class="surface">
      <EmptyState :title="t('page.placeholder')" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";

import EmptyState from "@/components/common/EmptyState.vue";

defineProps<{
  titleKey: string;
  subtitleKey?: string;
}>();

const { t } = useI18n();
</script>

