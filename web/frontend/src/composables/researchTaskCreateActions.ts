import { errorMessage, toISOString, type ResearchForm } from "@/composables/researchWorkspaceHelpers";
import { dataApi } from "@/services/api/data";
import { marketApi } from "@/services/api/market";
import type { CreateDataSyncTask } from "@/types/app";
import { normalizeSymbolInput } from "@/utils/marketSymbols";

type ResearchTaskCreateOptions = {
  closeCreateModal: () => void;
  form: ResearchForm;
  loadTasks: () => Promise<void>;
  message: {
    error: (content: string) => void;
    success: (content: string) => void;
  };
  t: (key: string) => string;
};

export async function createResearchDataSyncTask(options: ResearchTaskCreateOptions) {
  const request = createDataSyncTaskRequest(options.form);
  let instrumentExists = false;
  try {
    instrumentExists = await activeCatalogInstrumentExists(request.exchange, request.symbol);
  } catch {
    options.message.error(options.t("research.instrumentValidationFailed"));
    return;
  }
  if (!instrumentExists) {
    options.message.error(options.t("research.instrumentNotInCatalog"));
    return;
  }

  try {
    await dataApi.createTask(request);
    options.closeCreateModal();
    options.message.success(options.t("research.taskCreated"));
    await options.loadTasks();
  } catch (error) {
    options.message.error(errorMessage(error, options.t("research.taskCreateFailed")));
  }
}

function createDataSyncTaskRequest(form: ResearchForm): CreateDataSyncTask {
  return {
    exchange: form.exchange,
    symbol: normalizeSymbolInput(form.symbol),
    interval: form.interval,
    startTime: toISOString(form.startTime),
    endTime: toISOString(form.endTime),
  };
}

async function activeCatalogInstrumentExists(exchangeID: string, symbolValue: string) {
  const instruments = await marketApi.listInstruments({ exchange: exchangeID, limit: 1, q: symbolValue });
  return instruments.some((instrument) => instrument.exchange === exchangeID && instrument.symbol === symbolValue);
}
