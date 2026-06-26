<template>
  <div class="app-shell">
    <header class="app-header">
      <div class="app-header__main">
        <BrandMark />
        <TopNav />
      </div>
      <TopActions />
    </header>
    <main class="app-main">
      <RouterView />
    </main>
  </div>
</template>

<script setup lang="ts">
import { RouterView } from "vue-router";

import BrandMark from "@/components/layout/BrandMark.vue";
import TopActions from "@/components/layout/TopActions.vue";
import TopNav from "@/components/layout/TopNav.vue";
</script>

