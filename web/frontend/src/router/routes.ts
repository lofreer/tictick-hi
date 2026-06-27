import type { RouteRecordRaw } from "vue-router";

const AppShell = () => import("@/components/layout/AppShell.vue");
const LoginPage = () => import("@/pages/LoginPage.vue");
const OverviewPage = () => import("@/pages/OverviewPage.vue");
const ResearchPage = () => import("@/pages/ResearchPage.vue");
const BacktestsPage = () => import("@/pages/BacktestsPage.vue");
const BacktestNewPage = () => import("@/pages/BacktestNewPage.vue");
const BacktestDetailPage = () => import("@/pages/BacktestDetailPage.vue");
const TradingPage = () => import("@/pages/TradingPage.vue");
const TradingNewPage = () => import("@/pages/TradingNewPage.vue");
const TradingDetailPage = () => import("@/pages/TradingDetailPage.vue");
const SystemNotificationsPage = () => import("@/pages/SystemNotificationsPage.vue");
const SystemExchangeAccountsPage = () => import("@/pages/SystemExchangeAccountsPage.vue");
const SystemOperatorsPage = () => import("@/pages/SystemOperatorsPage.vue");
const SystemSessionsPage = () => import("@/pages/SystemSessionsPage.vue");
const SystemAuditEventsPage = () => import("@/pages/SystemAuditEventsPage.vue");
const SystemHealthPage = () => import("@/pages/SystemHealthPage.vue");

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
        component: OverviewPage,
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
        path: "system/sessions",
        name: "system-sessions",
        component: SystemSessionsPage,
      },
      {
        path: "system/audit-events",
        name: "system-audit-events",
        component: SystemAuditEventsPage,
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
