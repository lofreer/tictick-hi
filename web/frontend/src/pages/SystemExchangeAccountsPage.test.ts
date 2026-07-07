import { flushPromises, mount } from "@vue/test-utils";
import { NMessageProvider } from "naive-ui";
import { createPinia, setActivePinia, type Pinia } from "pinia";
import { defineComponent, h } from "vue";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { i18n } from "@/i18n";
import SystemExchangeAccountsPage from "@/pages/SystemExchangeAccountsPage.vue";
import { useAuthStore } from "@/stores/auth";
import type { ExchangeAccount } from "@/types/app";

const apiMocks = vi.hoisted(() => ({
  listExchangeAccounts: vi.fn(),
  createExchangeAccount: vi.fn(),
}));

vi.mock("@/services/api/system", () => ({
  systemApi: {
    listExchangeAccounts: apiMocks.listExchangeAccounts,
    createExchangeAccount: apiMocks.createExchangeAccount,
  },
}));

let pinia: Pinia;

describe("SystemExchangeAccountsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    pinia = createPinia();
    setActivePinia(pinia);
    useAuthStore().operator = operator("op_admin", "admin", "admin");
  });

  it("shows exchange account creation to admins", async () => {
    apiMocks.listExchangeAccounts.mockResolvedValue([]);

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).toContain("创建账号");
  });

  it("hides exchange account creation from non-admin operators", async () => {
    useAuthStore().operator = operator("op_ops", "ops", "operator");
    apiMocks.listExchangeAccounts.mockResolvedValue([account("ea_ops")]);

    const wrapper = mountPage();
    await flushPromises();

    expect(wrapper.text()).not.toContain("创建账号");
    expect(wrapper.text()).toContain("main");
  });
});

function mountPage() {
  const Host = defineComponent({
    render: () => h(NMessageProvider, null, { default: () => h(SystemExchangeAccountsPage) }),
  });

  return mount(Host, {
    global: {
      plugins: [i18n, pinia],
    },
  });
}

function account(id: string): ExchangeAccount {
  return {
    id,
    exchange: "binance",
    alias: "main",
    enabled: true,
    credentialStatus: "encrypted",
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
  };
}

function operator(id: string, username: string, role: string) {
  return {
    id,
    username,
    role,
    enabled: true,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
  };
}
