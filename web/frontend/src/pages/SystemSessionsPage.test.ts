import { flushPromises, mount } from "@vue/test-utils";
import { NMessageProvider } from "naive-ui";
import { defineComponent, h } from "vue";
import { describe, expect, it, vi } from "vitest";

import { i18n } from "@/i18n";
import SystemSessionsPage from "@/pages/SystemSessionsPage.vue";

const apiMocks = vi.hoisted(() => ({
  listSessions: vi.fn(),
  revokeSession: vi.fn(),
}));

vi.mock("@/services/api/auth", () => ({
  authApi: {
    listSessions: apiMocks.listSessions,
    revokeSession: apiMocks.revokeSession,
  },
}));

describe("SystemSessionsPage", () => {
  it("shows session source context", async () => {
    apiMocks.listSessions.mockResolvedValue([
      {
        id: "os_1",
        current: true,
        remoteAddr: "203.0.113.24",
        userAgent: "tictick-hi-test/1.0",
        createdAt: "2026-01-01T00:00:00Z",
        expiresAt: "2026-01-02T00:00:00Z",
      },
    ]);
    const Host = defineComponent({
      render: () => h(NMessageProvider, null, { default: () => h(SystemSessionsPage) }),
    });

    const wrapper = mount(Host, {
      global: {
        plugins: [i18n],
      },
    });
    await flushPromises();

    expect(wrapper.text()).toContain("来源地址");
    expect(wrapper.text()).toContain("203.0.113.24");
    expect(wrapper.text()).toContain("User-Agent");
    expect(wrapper.text()).toContain("tictick-hi-test/1.0");
    expect(wrapper.get(".session-user-agent").attributes("title")).toBe("tictick-hi-test/1.0");
  });
});
