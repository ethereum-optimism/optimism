import { HopeThemeConfig, ResolvedHopeThemeConfig } from "./theme";
import { SiteConfig } from "@mr-hope/vuepress-types";

export * from "./appearance";
export * from "./extends";
export * from "./feature";
export * from "./layout";
export * from "./locale";
export * from "./plugin";
export * from "./theme";

/** vuepress-theme-hope 项目配置 */
export interface HopeVuePressConfig extends SiteConfig {
  /** 自定义主题的配置 */
  themeConfig: HopeThemeConfig;
}

/** 处理过的 vuepress-theme-hope 项目配置 */
export interface ResolvedHopeVuePressConfig extends HopeVuePressConfig {
  /** 使用的自定义主题 */
  theme: "hope";
  /** 自定义主题的配置 */
  themeConfig: ResolvedHopeThemeConfig;
}
