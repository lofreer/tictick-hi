import { flushPromises, mount } from "@vue/test-utils";
import { NMessageProvider } from "naive-ui";
import { defineComponent, h } from "vue";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { i18n } from "@/i18n";
import SystemNotificationsPage from "@/pages/SystemNotificationsPage.vue";
import type { NotificationChannel } from "@/types/app";

const apiMocks = vi.hoisted(() => ({
  listNotifications: vi.fn(),
  listNotificationChannels: vi.fn(),
  setNotificationChannelEnabled: vi.fn(),
  createNotificationChannel: vi.fn(),
  retryNotification: vi.fn(),
}));

const routeMocks = vi.hoisted(() => ({
  query: {} as Record<string, string>,
  replace: vi.fn(),
}));

vi.mock("@/services/api/system", () => ({
  systemApi: {
    listNotifications: apiMocks.listNotifications,
    listNotificationChannels: apiMocks.listNotificationChannels,
    setNotificationChannelEnabled: apiMocks.setNotificationChannelEnabled,
    createNotificationChannel: apiMocks.createNotificationChannel,
    retryNotification: apiMocks.retryNotification,
  },
}));

vi.mock("vue-router", () => ({
  useRoute: () => ({ query: routeMocks.query }),
  useRouter: () => ({ replace: routeMocks.replace }),
}));

describe("SystemNotificationsPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    routeMocks.query = {};
  });

  it("toggles notification channel enabled state from the channel table", async () => {
    apiMocks.listNotifications.mockResolvedValue([]);
    apiMocks.listNotificationChannels
      .mockResolvedValueOnce([channel("nc_ops", true)])
      .mockResolvedValueOnce([channel("nc_ops", false)]);
    apiMocks.setNotificationChannelEnabled.mockResolvedValue(channel("nc_ops", false));

    const Host = defineComponent({
      render: () => h(NMessageProvider, null, { default: () => h(SystemNotificationsPage) }),
    });

    const wrapper = mount(Host, {
      global: {
        plugins: [i18n],
      },
    });
    await flushPromises();

    const rows = wrapper.findAll("tbody tr");
    expect(rows).toHaveLength(1);
    const button = rows[0].get("button");
    expect(button.text()).toContain("停用");

    await button.trigger("click");
    await flushPromises();

    expect(apiMocks.setNotificationChannelEnabled).toHaveBeenCalledWith("nc_ops", false);
    expect(apiMocks.listNotificationChannels).toHaveBeenCalledTimes(2);
    expect(wrapper.find("tbody tr").text()).toContain("否");
  });
});

function channel(id: string, enabled: boolean): NotificationChannel {
  return {
    id,
    name: "Ops",
    provider: "local",
    target: "default",
    enabled,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
  };
}
