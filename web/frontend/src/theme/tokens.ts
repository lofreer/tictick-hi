import type { GlobalThemeOverrides } from "naive-ui";

import type { ThemeMode } from "@/types/app";

export const appColors = {
  gold: "#e8a700",
  goldDark: "#ad7a00",
  success: "#0f9f6e",
  danger: "#d93a4a",
  warning: "#e88700",
  info: "#246b8f",
};

export const chartAxisFontSize = 13;
export const chartMobileAxisFontSize = 13;
export const chartRightPriceScaleWidth = {
  desktop: 56,
  narrowDesktop: 56,
  mobile: 54,
};

export const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: appColors.gold,
    primaryColorHover: "#f0b90b",
    primaryColorPressed: appColors.goldDark,
    primaryColorSuppl: "#f4c542",
    successColor: appColors.success,
    warningColor: appColors.warning,
    errorColor: appColors.danger,
    infoColor: appColors.info,
    borderRadius: "8px",
    borderRadiusSmall: "6px",
    fontFamily:
      "Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, Segoe UI, sans-serif",
  },
  Button: {
    borderRadiusMedium: "8px",
    fontWeightStrong: "650",
  },
  Card: {
    borderRadius: "8px",
    paddingMedium: "24px",
  },
  DataTable: {
    thFontWeight: "650",
    thColor: "var(--tt-surface-raised)",
    tdColorHover: "var(--tt-surface-raised)",
  },
  Tag: {
    borderRadius: "6px",
  },
};

export function applyThemeAttribute(mode: ThemeMode) {
  document.documentElement.dataset.theme = mode;
  document.documentElement.style.colorScheme = mode;
}

export function chartTheme(mode: ThemeMode) {
  const dark = mode === "dark";

  return {
    layout: {
      background: { color: dark ? "#181a20" : "#ffffff" },
      fontFamily: "Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, Segoe UI, sans-serif",
      fontSize: chartAxisFontSize,
      textColor: dark ? "#b7bdc6" : "#474d57",
    },
    grid: {
      vertLines: { color: dark ? "#252930" : "#eff0f2" },
      horzLines: { color: dark ? "#252930" : "#eff0f2" },
    },
    rightPriceScale: {
      borderColor: dark ? "#2b3139" : "#eaecef",
      minimumWidth: chartRightPriceScaleWidth.desktop,
    },
    timeScale: {
      borderColor: dark ? "#2b3139" : "#eaecef",
      timeVisible: true,
    },
    crosshair: {
      vertLine: { color: appColors.info },
      horzLine: { color: appColors.info },
    },
  };
}
