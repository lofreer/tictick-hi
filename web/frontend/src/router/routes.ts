import type { RouteRecordRaw } from "vue-router";

import AppShell from "@/components/layout/AppShell.vue";
import BacktestDetailPage from "@/pages/BacktestDetailPage.vue";
import BacktestNewPage from "@/pages/BacktestNewPage.vue";
import BacktestsPage from "@/pages/BacktestsPage.vue";
import LoginPage from "@/pages/LoginPage.vue";
import PageStub from "@/pages/PageStub.vue";
import ResearchPage from "@/pages/ResearchPage.vue";
import SystemExchangeAccountsPage from "@/pages/SystemExchangeAccountsPage.vue";
import SystemHealthPage from "@/pages/SystemHealthPage.vue";
import SystemNotificationsPage from "@/pages/SystemNotificationsPage.vue";
import SystemOperatorsPage from "@/pages/SystemOperatorsPage.vue";
import TradingDetailPage from "@/pages/TradingDetailPage.vue";
import TradingNewPage from "@/pages/TradingNewPage.vue";
import TradingPage from "@/pages/TradingPage.vue";

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
        component: TradingPage,
      },
      {
        path: "trading/new",
        name: "trading-new",
        component: TradingNewPage,
      },
      {
        path: "trading/:id",
        name: "trading-detail",
        component: TradingDetailPage,
      },
      {
        path: "system/notifications",
        name: "system-notifications",
        component: SystemNotificationsPage,
      },
      {
        path: "system/exchange-accounts",
        name: "system-exchange-accounts",
        component: SystemExchangeAccountsPage,
      },
      {
        path: "system/operators",
        name: "system-operators",
        component: SystemOperatorsPage,
      },
      {
        path: "system/health",
        name: "system-health",
        component: SystemHealthPage,
      },
    ],
  },
  { path: "/:pathMatch(.*)*", redirect: "/overview" },
];
