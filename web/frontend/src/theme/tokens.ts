import type { GlobalThemeOverrides } from "naive-ui";

import type { ThemeMode } from "@/types/app";

export const appColors = {
  gold: "#f0b90b",
  goldDark: "#c09409",
  success: "#0ecb81",
  danger: "#f6465d",
  warning: "#f7a600",
  info: "#848e9c",
};

export const chartAxisFontSize = 42;
export const chartMobileAxisFontSize = 32;
export const chartRightPriceScaleWidth = {
  desktop: 158,
  narrowDesktop: 150,
  mobile: 118,
};

export const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: appColors.gold,
    primaryColorHover: "#f2c53d",
    primaryColorPressed: appColors.goldDark,
    primaryColorSuppl: "#f5d165",
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
  },
  DataTable: {
    thFontWeight: "650",
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
