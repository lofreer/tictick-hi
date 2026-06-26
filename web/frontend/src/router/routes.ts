import type { RouteRecordRaw } from "vue-router";

import AppShell from "@/components/layout/AppShell.vue";
import LoginPage from "@/pages/LoginPage.vue";
import PageStub from "@/pages/PageStub.vue";
import ResearchPage from "@/pages/ResearchPage.vue";

export const routes: RouteRecordRaw[] = [
  {
    path: "/login",
    name: "login",
    component: LoginPage,
    meta: { public: true },
  },
  {
    path: "/",
    component: AppShell,
    meta: { requiresAuth: true },
    children: [
      { path: "", redirect: "/overview" },
      {
        path: "overview",
        name: "overview",
        component: PageStub,
        props: { titleKey: "page.overview.title", subtitleKey: "page.overview.subtitle" },
      },
      {
        path: "research",
        name: "research",
        component: ResearchPage,
      },
      {
        path: "backtests",
        name: "backtests",
        component: PageStub,
        props: { titleKey: "page.backtests.title", subtitleKey: "page.backtests.subtitle" },
      },
      {
        path: "backtests/new",
        name: "backtests-new",
        component: PageStub,
        props: { titleKey: "page.backtestsNew.title" },
      },
      {
        path: "backtests/:id",
        name: "backtests-detail",
        component: PageStub,
        props: { titleKey: "page.backtestsDetail.title" },
      },
      {
        path: "trading",
        name: "trading",
        component: PageStub,
        props: { titleKey: "page.trading.title", subtitleKey: "page.trading.subtitle" },
      },
      {
        path: "trading/new",
        name: "trading-new",
        component: PageStub,
        props: { titleKey: "page.tradingNew.title" },
      },
      {
        path: "trading/:id",
        name: "trading-detail",
        component: PageStub,
        props: { titleKey: "page.tradingDetail.title" },
      },
      {
        path: "system/notifications",
        name: "system-notifications",
        component: PageStub,
        props: { titleKey: "page.notifications.title" },
      },
      {
        path: "system/exchange-accounts",
        name: "system-exchange-accounts",
        component: PageStub,
        props: { titleKey: "page.exchangeAccounts.title" },
      },
      {
        path: "system/operators",
        name: "system-operators",
        component: PageStub,
        props: { titleKey: "page.operators.title" },
      },
      {
        path: "system/health",
        name: "system-health",
        component: PageStub,
        props: { titleKey: "page.health.title" },
      },
    ],
  },
  { path: "/:pathMatch(.*)*", redirect: "/overview" },
];

