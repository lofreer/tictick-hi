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
  revokeOperatorSessions: vi.fn(),
  createOperator: vi.fn(),
}));

vi.mock("@/services/api/system", () => ({
  systemApi: {
    listOperators: apiMocks.listOperators,
    setOperatorEnabled: apiMocks.setOperatorEnabled,
    setOperatorRole: apiMocks.setOperatorRole,
    revokeOperatorSessions: apiMocks.revokeOperatorSessions,
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
    expect(selfButtons).toHaveLength(3);
    const selfRoleButton = selfButtons[0];
    const selfRevokeSessionsButton = selfButtons[1];
    const selfDisableButton = selfButtons[2];
    expect(selfRoleButton.attributes("disabled")).toBeDefined();
    expect(selfRoleButton.attributes("title")).toBe("不能变更当前操作员角色。");
    await selfRoleButton.trigger("click");
    expect(apiMocks.setOperatorRole).not.toHaveBeenCalled();
    expect(selfRevokeSessionsButton.attributes("disabled")).toBeDefined();
    expect(selfRevokeSessionsButton.attributes("title")).toBe("不能在这里撤销当前操作员会话。");
    await selfRevokeSessionsButton.trigger("click");
    expect(apiMocks.revokeOperatorSessions).not.toHaveBeenCalled();
    expect(selfDisableButton.attributes("disabled")).toBeDefined();
    expect(selfDisableButton.attributes("title")).toBe("不能停用当前操作员。");
    await selfDisableButton.trigger("click");
    expect(apiMocks.setOperatorEnabled).not.toHaveBeenCalled();

    const otherButtons = rows[1].findAll("button");
    expect(otherButtons).toHaveLength(3);
    expect(otherButtons[0].attributes("disabled")).toBeUndefined();
    expect(otherButtons[1].attributes("disabled")).toBeUndefined();
    expect(otherButtons[2].attributes("disabled")).toBeUndefined();
    await otherButtons[2].trigger("click");
    expect(apiMocks.setOperatorEnabled).toHaveBeenCalledWith("op_ops", false);
  });

  it("revokes another operator's sessions from the admin UI", async () => {
    apiMocks.listOperators.mockResolvedValue([
      operator("op_admin", "admin", true),
      operator("op_ops", "ops", true, "operator"),
    ]);
    apiMocks.revokeOperatorSessions.mockResolvedValue({ revokedSessionCount: 2 });
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
    const otherButtons = rows[1].findAll("button");
    await otherButtons[1].trigger("click");

    expect(apiMocks.revokeOperatorSessions).toHaveBeenCalledWith("op_ops");
  });

  it("hides operator management actions from non-admin operators", async () => {
    apiMocks.listOperators.mockResolvedValue([
      operator("op_admin", "admin", true),
      operator("op_ops", "ops", true, "operator"),
    ]);
    const pinia = createPinia();
    setActivePinia(pinia);
    useAuthStore().operator = operator("op_ops", "ops", true, "operator");

    const Host = defineComponent({
      render: () => h(NMessageProvider, null, { default: () => h(SystemOperatorsPage) }),
    });

    const wrapper = mount(Host, {
      global: {
        plugins: [i18n, pinia],
      },
    });
    await flushPromises();

    expect(wrapper.text()).not.toContain("创建操作员");
    const rows = wrapper.findAll("tbody tr");
    expect(rows).toHaveLength(2);
    expect(rows[0].findAll("button")).toHaveLength(0);
    expect(rows[1].findAll("button")).toHaveLength(0);
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
