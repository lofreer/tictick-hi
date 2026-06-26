<template>
  <NForm class="strategy-param-form" label-placement="top" :disabled="disabled">
    <div class="strategy-param-form__grid">
      <NFormItem
        v-for="param in params"
        :key="param.key"
        :label="param.label"
        :feedback="param.description"
      >
        <NInputNumber
          v-if="param.type === 'number'"
          class="strategy-param-form__control"
          :value="numberValue(param.key)"
          :min="param.min"
          :max="param.max"
          :step="param.step"
          @update:value="(value) => setValue(param.key, value ?? null)"
        />
        <NSelect
          v-else-if="param.type === 'select'"
          class="strategy-param-form__control"
          :value="stringValue(param.key)"
          :options="selectOptions(param)"
          @update:value="(value) => setValue(param.key, value === null ? '' : String(value))"
        />
        <NSwitch
          v-else-if="param.type === 'boolean'"
          :value="booleanValue(param.key)"
          @update:value="(value) => setValue(param.key, value)"
        />
        <NInput
          v-else
          class="strategy-param-form__control"
          :value="stringValue(param.key)"
          @update:value="(value) => setValue(param.key, value)"
        />
      </NFormItem>
    </div>
  </NForm>
</template>

<script setup lang="ts">
import {
  NForm,
  NFormItem,
  NInput,
  NInputNumber,
  NSelect,
  NSwitch,
  type SelectOption,
} from "naive-ui";

import type { StrategyParamSpec, StrategyParamValue, StrategyParamValues } from "@/types/app";

const props = defineProps<{
  disabled?: boolean;
  params: StrategyParamSpec[];
  value: StrategyParamValues;
}>();

const emit = defineEmits<{
  "update:value": [value: StrategyParamValues];
}>();

function setValue(key: string, value: StrategyParamValue) {
  emit("update:value", { ...props.value, [key]: value });
}

function numberValue(key: string) {
  const value = props.value[key];
  return typeof value === "number" ? value : null;
}

function stringValue(key: string) {
  const value = props.value[key];
  return typeof value === "string" || typeof value === "number" ? String(value) : "";
}

function booleanValue(key: string) {
  return props.value[key] === true;
}

function selectOptions(param: StrategyParamSpec): SelectOption[] {
  return param.options.map((option) => ({ label: option.label, value: option.value }));
}
</script>

<style scoped>
.strategy-param-form__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 14px;
}

.strategy-param-form__control {
  width: 100%;
}

@media (max-width: 720px) {
  .strategy-param-form__grid {
    grid-template-columns: 1fr;
  }
}
</style>
