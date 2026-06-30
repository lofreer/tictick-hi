<template>
  <NModal v-model:show="modalOpen" preset="card" :title="t('research.gapDetailsTitle')" class="research-modal">
    <div v-if="task" class="research-gap-context">
      <NText depth="3">{{ task.exchange }} / {{ task.symbol }} / {{ task.interval }}</NText>
    </div>
    <LoadingState v-if="loading" />
    <ErrorState v-else-if="error" :title="error" retryable @retry="emit('retry')" />
    <EmptyState v-else-if="!details || details.gaps.length === 0" :title="t('research.noGapDetails')" />
    <NDataTable v-else :columns="columns" :data="details.gaps" :bordered="false" size="small" />
    <template #footer>
      <NSpace align="center" justify="space-between">
        <NSpace align="center">
          <NTag v-if="details?.limited" :bordered="false" type="warning">
            {{
              t("research.gapDetailsLimited", {
                returned: details.returnedCount,
                total: details.totalCount,
                limit: details.repairLimit,
              })
            }}
          </NTag>
          <NText v-if="repairNotice" :type="repairNoticeType">{{ repairNotice }}</NText>
          <MarketRepairResultTags :result="repairResult" :tasks="tasks" />
        </NSpace>
        <NSpace align="center" justify="end">
          <NButton
            v-if="task && details && details.totalCount > 0"
            :loading="repairLoading"
            secondary
            size="small"
            type="warning"
            @click="emit('repair')"
          >
            {{ t("research.repairTaskGaps") }}
          </NButton>
          <NButton @click="modalOpen = false">{{ t("common.close") }}</NButton>
        </NSpace>
      </NSpace>
    </template>
  </NModal>
</template>

<script setup lang="ts">
import { NButton, NDataTable, NModal, NSpace, NTag, NText, type DataTableColumns } from "naive-ui";
import { computed } from "vue";
import { useI18n } from "vue-i18n";

import EmptyState from "@/components/common/EmptyState.vue";
import ErrorState from "@/components/common/ErrorState.vue";
import LoadingState from "@/components/common/LoadingState.vue";
import MarketRepairResultTags from "@/components/research/MarketRepairResultTags.vue";
import type { CandleGap, DataSyncGapList, DataSyncGapRepairResult, DataSyncTask } from "@/types/app";

const props = defineProps<{
  details: DataSyncGapList | null;
  error: string;
  loading: boolean;
  repairLoading: boolean;
  repairNotice: string;
  repairNoticeType: "success" | "error" | "warning" | "default";
  repairResult: DataSyncGapRepairResult | null;
  show: boolean;
  task: DataSyncTask | null;
  tasks?: DataSyncTask[];
}>();

const emit = defineEmits<{
  repair: [];
  retry: [];
  "update:show": [value: boolean];
}>();

const { t } = useI18n();
const modalOpen = computed({
  get: () => props.show,
  set: (value: boolean) => emit("update:show", value),
});
const columns = computed<DataTableColumns<CandleGap>>(() => [
  { title: t("research.gapFrom"), key: "from", minWidth: 180 },
  { title: t("research.gapTo"), key: "to", minWidth: 180 },
  { title: t("research.missingCandles"), key: "missingCandles", width: 120 },
]);
</script>
