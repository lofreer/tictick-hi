import type { Ref } from "vue";

import { dataApi } from "@/services/api/data";
import type { DataSyncTask } from "@/types/app";

type ResearchTaskSnapshotRefs = {
  gapDetailsTask: Ref<DataSyncTask | null>;
  selectedChartTask: Ref<DataSyncTask | null>;
  tasks: Ref<DataSyncTask[]>;
};

export async function loadResearchTaskSnapshots(ids: string[], refs: ResearchTaskSnapshotRefs) {
  const uniqueIds = [...new Set(ids.filter(Boolean))];
  if (uniqueIds.length === 0) return [];
  const snapshots = (await Promise.all(uniqueIds.map(loadTaskSnapshot))).filter((task): task is DataSyncTask => task !== null);
  mergeTaskSnapshots(snapshots, refs);
  return snapshots;
}

async function loadTaskSnapshot(id: string) {
  try {
    return await dataApi.getTask(id);
  } catch {
    return null;
  }
}

function mergeTaskSnapshots(snapshots: DataSyncTask[], refs: ResearchTaskSnapshotRefs) {
  if (snapshots.length === 0) return;
  const snapshotsByID = new Map(snapshots.map((task) => [task.id, task]));
  const existingIDs = new Set(refs.tasks.value.map((task) => task.id));
  refs.tasks.value = [
    ...refs.tasks.value.map((task) => snapshotsByID.get(task.id) ?? task),
    ...snapshots.filter((task) => !existingIDs.has(task.id)),
  ];
  if (refs.selectedChartTask.value) {
    refs.selectedChartTask.value = snapshotsByID.get(refs.selectedChartTask.value.id) ?? refs.selectedChartTask.value;
  }
  if (refs.gapDetailsTask.value) {
    refs.gapDetailsTask.value = snapshotsByID.get(refs.gapDetailsTask.value.id) ?? refs.gapDetailsTask.value;
  }
}
