import type { RouteRecordRaw } from "vue-router";

import AppShell from "@/components/layout/AppShell.vue";
import BacktestDetailPage from "@/pages/BacktestDetailPage.vue";
import BacktestNewPage from "@/pages/BacktestNewPage.vue";
import BacktestsPage from "@/pages/BacktestsPage.vue";
import LoginPage from "@/pages/LoginPage.vue";
import PageStub from "@/pages/PageStub.vue";
import ResearchPage from "@/pages/ResearchPage.vue";
import TradingNewPage from "@/pages/TradingNewPage.vue";

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
        component: BacktestsPage,
      },
      {
        path: "backtests/new",
        name: "backtests-new",
        component: BacktestNewPage,
      },
      {
        path: "backtests/:id",
        name: "backtests-detail",
        component: BacktestDetailPage,
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
        component: TradingNewPage,
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
