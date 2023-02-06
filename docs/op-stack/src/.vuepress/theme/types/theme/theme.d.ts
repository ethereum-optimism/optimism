import { AlgoliaOption, I18nConfig } from "@mr-hope/vuepress-types";
import { HopeThemeAppearanceConfig } from "./appearance";
import { HopeFeatureConfig } from "./feature";
import { HopeLayoutConfig } from "./layout";
import { HopeLangLocalesConfig } from "./locale";
import { HopeThemePluginConfig } from "./plugin";

/** vuepress-theme-hope 主题配置 */
export interface HopeThemeConfig
  extends HopeThemeAppearanceConfig,
    HopeFeatureConfig,
    HopeLayoutConfig,
    HopeThemePluginConfig {
  /** 导航栏 Logo，应为绝对路径 */
  logo?: string;
  /**
   * 暗黑模式下 logo
   *
   * Logo Image under darkmode
   */
  darkLogo?: string;
  /** 显示所有页面的标题链接 */
  displayAllHeaders?: boolean;
  /** 是否启用默认的搜索框 */
  search?: boolean;
  /** 搜索框占位符 */
  searchPlaceholder?: string;
  /** 默认搜索框显示的搜索结果数量 */
  searchMaxSuggestions?: number;
  /** Algolia 搜索配置 */
  algolia?: AlgoliaOption;
  /** 所有页面的 下一篇 链接 */
  nextLinks?: boolean;
  /** 所有页面的 上一篇 链接 */
  prevLinks?: boolean;
  /** 项目仓库地址 */
  repo?: string;
  /** 仓库标签文字 */
  repoLabel?: string;
  /** 文档所属仓库 */
  docsRepo?: string;
  /** 文档所属文件夹 */
  docsDir?: string;
  /** 文档所属分支 */
  docsBranch?: string;

  /**
   * 多语言配置
   *
   * i18n config
   */
  locales?: Record<string, I18nConfig & HopeLangLocalesConfig>;

  /** 站点地址 */
  hostname?: string;

  /**
   * 文章显示的默认作者
   *
   * The default author of the article
   */
  author?: string;
}

/** 处理后的 vuepress-theme-hope 主题配置 */
export interface ResolvedHopeThemeConfig extends HopeThemeConfig {
  /** 侧边栏深度 */
  sidebarDepth: number;
  /** 图标 FontClass 前缀 */
  iconPrefix: string;
  /** 多语言配置 */
  locales: Record<string, I18nConfig & HopeLangLocalesConfig>;
  /** 页脚配置 */
  footer: HopeFooterConfig;
  /** 显示编辑本页链接 */
  editLinks: boolean;
}
