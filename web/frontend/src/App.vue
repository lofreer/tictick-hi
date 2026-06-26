<template>
  <AppProviders>
    <RouterView />
  </AppProviders>
</template>

<script setup lang="ts">
import { RouterView } from "vue-router";

import AppProviders from "@/app/AppProviders.vue";
</script>

