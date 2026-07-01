<template>
  <NButton
    v-if="canRepair"
    size="tiny"
    secondary
    type="error"
    :loading="repairLoading"
    @click="repairInvalidIssue"
  >
    {{ t("research.repairFirstInvalidIssue") }}
  </NButton>
  <MarketRepairResultTags :result="repairResult" :tasks="tasks" />
</template>

<script setup lang="ts">
import { NButton, useMessage } from "naive-ui";
import { computed, ref, watch } from "vue";
import { useI18n } from "vue-i18n";

import { repairChartInvalidIssue } from "@/composables/researchInvalidIssueRepairActions";
import { errorMessage } from "@/composables/researchWorkspaceHelpers";
import type { CandleIssue, DataSyncGapRepairResult, DataSyncTask } from "@/types/app";
import { normalizeSymbolInput } from "@/utils/marketSymbols";
import MarketRepairResultTags from "./MarketRepairResultTags.vue";

const props = defineProps<{
  exchange: string;
  interval: string;
  issue: CandleIssue | null;
  loadCandles: () => Promise<void>;
  loadTasks: () => Promise<void>;
  symbol: string;
  tasks: DataSyncTask[];
}>();

const emit = defineEmits<{
  repaired: [];
}>();

const { t } = useI18n();
const message = useMessage();
const repairLoading = ref(false);
const repairResult = ref<DataSyncGapRepairResult | null>(null);
const canRepair = computed(() => Boolean(props.issue?.openTime));

watch(
  () => [props.exchange, props.symbol, props.interval, props.issue?.openTime],
  () => {
    repairResult.value = null;
  },
);

async function repairInvalidIssue() {
  if (!props.issue?.openTime) return;
  repairLoading.value = true;
  try {
    repairResult.value = await repairChartInvalidIssue({
      exchange: props.exchange,
      interval: props.interval,
      issue: props.issue,
      loadCandles: props.loadCandles,
      loadTasks: props.loadTasks,
      onSuccess: (messageKey, values) => message.success(t(messageKey, values ?? {})),
      symbol: normalizeSymbolInput(props.symbol),
    });
    emit("repaired");
  } catch (error) {
    message.error(errorMessage(error, t("research.invalidIssueRepairFailed")));
  } finally {
    repairLoading.value = false;
  }
}
</script>
