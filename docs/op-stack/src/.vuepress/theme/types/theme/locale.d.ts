import {
  HopeNavBarConfig,
  HopeSideBarConfig,
  HopeThemeLocaleConfigItem,
} from "@mr-hope/vuepress-shared";
import { AlgoliaOption } from "@mr-hope/vuepress-types";
import { HopeFooterConfig } from "./layout";

/** vuepress-theme-hope 多语言配置 */
export interface HopeLangLocalesConfig
  extends Partial<HopeThemeLocaleConfigItem> {
  /** 当前语言下的标题 */
  title?: string;
  /** 当前语言下的描述 */
  description?: string;
  /** 导航栏链接 */
  nav?: HopeNavBarConfig;
  /** 侧边栏配置 */
  sidebar?: HopeSideBarConfig;
  /** 当前语言的 algolia 设置 */
  algolia?: AlgoliaOption;
  /** 页脚设置 */
  footer?: HopeFooterConfig;
}
