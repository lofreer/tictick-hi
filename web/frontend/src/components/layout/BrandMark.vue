<template>
  <RouterLink class="brand-mark" to="/overview" :aria-label="$t('app.name')">
    <svg class="brand-mark__icon" width="36" height="36" viewBox="0 0 36 36" fill="none" aria-hidden="true">
      <rect class="brand-mark__box" width="36" height="36" rx="8" />
      <path
        d="M7 22 L14 28 L29 9"
        stroke="var(--tt-logo-stroke)"
        stroke-width="4"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
    </svg>
    <span class="brand-mark__text">tict<span>ı</span>ck-hi</span>
  </RouterLink>
</template>

<script setup lang="ts">
import { RouterLink } from "vue-router";
</script>

