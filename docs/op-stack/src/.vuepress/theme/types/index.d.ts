import {
  HopeNavBarConfig,
  HopeSideBarConfig,
  HopeThemeConfig,
  HopeVuePressConfig,
  ResolvedHopeVuePressConfig,
} from "./theme";
import "./declare";
import "./extend";

export * from "./theme";

export const config: (config: HopeVuePressConfig) => ResolvedHopeVuePressConfig;

export const themeConfig: (themeConfig: HopeThemeConfig) => HopeThemeConfig;
export const navbarConfig: (navbarConfig: HopeNavBarConfig) => HopeNavBarConfig;
export const sidebarConfig: (
  sidebarConfig: HopeSideBarConfig
) => HopeSideBarConfig;
