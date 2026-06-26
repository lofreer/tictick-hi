<template>
  <nav class="top-nav" aria-label="Primary">
    <RouterLink v-for="item in navItems" :key="item.to" class="top-nav__link" :to="item.to">
      {{ t(item.labelKey) }}
    </RouterLink>
  </nav>
</template>

<script setup lang="ts">
import { RouterLink } from "vue-router";
import { useI18n } from "vue-i18n";

const { t } = useI18n();

const navItems = [
  { to: "/overview", labelKey: "nav.overview" },
  { to: "/research", labelKey: "nav.research" },
  { to: "/backtests", labelKey: "nav.backtests" },
  { to: "/trading", labelKey: "nav.trading" },
];
</script>

