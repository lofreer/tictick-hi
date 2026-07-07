import { mount } from "@vue/test-utils";
import { createPinia, setActivePinia, type Pinia } from "pinia";
import { beforeEach, describe, expect, it, vi } from "vitest";

import SystemMenu from "@/components/layout/SystemMenu.vue";
import { useAuthStore } from "@/stores/auth";

const routerPush = vi.fn();

vi.mock("vue-router", () => ({
  useRouter: () => ({ push: routerPush }),
}));

vi.mock("vue-i18n", () => ({
  useI18n: () => ({ t: (key: string) => key }),
}));

vi.mock("naive-ui", () => ({
  NButton: {
    name: "NButton",
    template: "<button><slot name='icon' /><slot /></button>",
  },
  NDropdown: {
    name: "NDropdown",
    props: ["options"],
    emits: ["select"],
    template: `
      <nav>
        <button
          v-for="option in options"
          :key="option.key"
          type="button"
          @click="$emit('select', option.key)"
        >
          {{ option.label }}
        </button>
        <slot />
      </nav>
    `,
  },
}));

let pinia: Pinia;

describe("SystemMenu", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    pinia = createPinia();
    setActivePinia(pinia);
  });

  it("shows sensitive system management entries to admins", () => {
    useAuthStore().operator = operator("op_admin", "admin", "admin");
    const wrapper = mountMenu();

    expect(wrapper.text()).toContain("system.notifications");
    expect(wrapper.text()).toContain("system.exchangeAccounts");
    expect(wrapper.text()).toContain("system.operators");
  });

  it("hides sensitive system management entries from non-admin operators", () => {
    useAuthStore().operator = operator("op_ops", "ops", "operator");
    const wrapper = mountMenu();

    expect(wrapper.text()).not.toContain("system.notifications");
    expect(wrapper.text()).not.toContain("system.exchangeAccounts");
    expect(wrapper.text()).not.toContain("system.operators");
    expect(wrapper.text()).toContain("system.sessions");
  });
});

function mountMenu() {
  return mount(SystemMenu, {
    global: {
      plugins: [pinia],
    },
  });
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
