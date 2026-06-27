import { apiClient } from "@/services/api/client";
import type {
  StrategyDefinition as StrategyDefinitionResponse,
  StrategyParamSpec as StrategyParamResponse,
} from "@/types/api.generated";
import type { StrategyDefinition, StrategyParamSpec, StrategyParamValue } from "@/types/app";

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
    type: normalizeParamType(response.type),
    required: response.required,
    default: normalizeParamValue(response.default),
    min: response.min,
    max: response.max,
    step: response.step,
    options: response.options ?? [],
    description: response.description,
  };
}

function normalizeParamType(value: string): StrategyParamSpec["type"] {
  if (value === "number" || value === "select" || value === "text" || value === "boolean") {
    return value;
  }
  return "text";
}

function normalizeParamValue(value: unknown): StrategyParamValue | undefined {
  if (value === undefined) return undefined;
  if (value === null || typeof value === "string" || typeof value === "number" || typeof value === "boolean") {
    return value;
  }
  return undefined;
}
