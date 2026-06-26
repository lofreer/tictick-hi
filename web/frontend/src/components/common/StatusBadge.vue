<template>
  <NTag :bordered="false" :type="tagType" size="small">
    {{ t(`status.${status}`) }}
  </NTag>
</template>

<script setup lang="ts">
import { NTag, type TagProps } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import type { TaskStatus } from "@/types/app";

const props = defineProps<{ status: TaskStatus }>();
const { t } = useI18n();

const tagType = computed<TagProps["type"]>(() => {
  if (props.status === "running" || props.status === "succeeded") return "success";
  if (props.status === "failed" || props.status === "cancelled") return "error";
  if (props.status === "gap" || props.status === "stopping") return "warning";
  return "default";
});
</script>

