import type { Context, PluginOptionAPI } from "@mr-hope/vuepress-types";
import type { HopeNavBarConfig, HopeSideBarConfig, HopeThemeConfig, ResolvedHopeThemeConfig } from "./types";
declare const themeAPI: {
    (themeConfig: ResolvedHopeThemeConfig, ctx: Context): PluginOptionAPI;
    config: (config: import("./types").HopeVuePressConfig) => import("./types").ResolvedHopeVuePressConfig;
    themeConfig(themeConfig: HopeThemeConfig): HopeThemeConfig;
    navbarConfig(navbarConfig: HopeNavBarConfig): HopeNavBarConfig;
    sidebarConfig(sidebarConfig: HopeSideBarConfig): HopeSideBarConfig;
};
export = themeAPI;
