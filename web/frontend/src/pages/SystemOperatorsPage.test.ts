import { flushPromises, mount } from "@vue/test-utils";
import { NMessageProvider } from "naive-ui";
import { createPinia, setActivePinia } from "pinia";
import { defineComponent, h } from "vue";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { i18n } from "@/i18n";
import SystemOperatorsPage from "@/pages/SystemOperatorsPage.vue";
import { useAuthStore } from "@/stores/auth";

const apiMocks = vi.hoisted(() => ({
  listOperators: vi.fn(),
  setOperatorEnabled: vi.fn(),
  setOperatorRole: vi.fn(),
  createOperator: vi.fn(),
}));

vi.mock("@/services/api/system", () => ({
  systemApi: {
    listOperators: apiMocks.listOperators,
    setOperatorEnabled: apiMocks.setOperatorEnabled,
    setOperatorRole: apiMocks.setOperatorRole,
    createOperator: apiMocks.createOperator,
  },
}));

describe("SystemOperatorsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    const pinia = createPinia();
    setActivePinia(pinia);
  });

  it("blocks disabling the current operator in the UI", async () => {
    apiMocks.listOperators.mockResolvedValue([
      operator("op_admin", "admin", true),
      operator("op_ops", "ops", true, "operator"),
    ]);
    apiMocks.setOperatorEnabled.mockResolvedValue(operator("op_ops", "ops", false, "operator"));
    const pinia = createPinia();
    setActivePinia(pinia);
    useAuthStore().operator = operator("op_admin", "admin", true);

    const Host = defineComponent({
      render: () => h(NMessageProvider, null, { default: () => h(SystemOperatorsPage) }),
    });

    const wrapper = mount(Host, {
      global: {
        plugins: [i18n, pinia],
      },
    });
    await flushPromises();

    const rows = wrapper.findAll("tbody tr");
    expect(rows).toHaveLength(2);
    expect(rows[0].text()).toContain("管理员");
    expect(rows[1].text()).toContain("操作员");
    const selfButtons = rows[0].findAll("button");
    expect(selfButtons).toHaveLength(2);
    const selfRoleButton = selfButtons[0];
    const selfDisableButton = selfButtons[1];
    expect(selfRoleButton.attributes("disabled")).toBeDefined();
    expect(selfRoleButton.attributes("title")).toBe("不能变更当前操作员角色。");
    await selfRoleButton.trigger("click");
    expect(apiMocks.setOperatorRole).not.toHaveBeenCalled();
    expect(selfDisableButton.attributes("disabled")).toBeDefined();
    expect(selfDisableButton.attributes("title")).toBe("不能停用当前操作员。");
    await selfDisableButton.trigger("click");
    expect(apiMocks.setOperatorEnabled).not.toHaveBeenCalled();

    const otherButtons = rows[1].findAll("button");
    expect(otherButtons).toHaveLength(2);
    expect(otherButtons[0].attributes("disabled")).toBeUndefined();
    expect(otherButtons[1].attributes("disabled")).toBeUndefined();
    await otherButtons[1].trigger("click");
    expect(apiMocks.setOperatorEnabled).toHaveBeenCalledWith("op_ops", false);
  });
});

function operator(id: string, username: string, enabled: boolean, role = "admin") {
  return {
    id,
    username,
    role,
    enabled,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
  };
}
