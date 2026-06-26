import { apiClient } from "@/services/api/client";
import type { StrategyDefinition, StrategyParamSpec } from "@/types/app";

type StrategyDefinitionResponse = Omit<StrategyDefinition, "params"> & {
  params?: StrategyParamResponse[];
};

type StrategyParamResponse = Omit<StrategyParamSpec, "options"> & {
  options?: StrategyParamSpec["options"];
};

export const strategiesApi = {
  async listStrategies() {
    const response = await apiClient.get<StrategyDefinitionResponse[]>("/strategies");
    return response.map(normalizeStrategy);
  },

  async getStrategy(id: string) {
    const response = await apiClient.get<StrategyDefinitionResponse>(`/strategies/${encodeURIComponent(id)}`);
    return normalizeStrategy(response);
  },
};

function normalizeStrategy(response: StrategyDefinitionResponse): StrategyDefinition {
  return {
    id: response.id,
    name: response.name,
    version: response.version,
    description: response.description,
    supportedIntervals: response.supportedIntervals ?? [],
    supportedIntents: response.supportedIntents ?? [],
    params: (response.params ?? []).map(normalizeParam),
  };
}

function normalizeParam(response: StrategyParamResponse): StrategyParamSpec {
  return {
    key: response.key,
    label: response.label,
    type: response.type,
    required: response.required,
    default: response.default,
    min: response.min,
    max: response.max,
    step: response.step,
    options: response.options ?? [],
    description: response.description,
  };
}
