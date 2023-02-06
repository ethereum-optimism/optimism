/**
 * 合法的媒体
 *
 * media you can choose
 */
type BlogMedia =
  | "Baidu"
  | "Bitbucket"
  | "Dingding"
  | "Discord"
  | "Dribbble"
  | "Email"
  | "Evernote"
  | "Facebook"
  | "Flipboard"
  | "Gitee"
  | "Github"
  | "Gitlab"
  | "Gmail"
  | "Instagram"
  | "Lines"
  | "Linkedin"
  | "Pinterest"
  | "Pocket"
  | "QQ"
  | "Qzone"
  | "Reddit"
  | "Rss"
  | "Steam"
  | "Twitter"
  | "Wechat"
  | "Weibo"
  | "Whatsapp"
  | "Youtube"
  | "Zhihu";

/**
 * 博客选项
 *
 * Blog configuration
 */
export type BlogOptions = {
  /**
   * 博主名称
   *
   * Name of the Blogger, default is author
   */
  name?: string;

  /**
   * 博主头像，应为绝对路径
   *
   * Blogger avator, must be an absolute path
   */
  avatar?: string;

  /**
   * 博主的个人介绍地址
   *
   * Intro page about blogger
   */
  intro?: string;

  /**
   * 媒体链接配置
   *
   * Media links configuration
   *
   * E.g.
   *
   * ```js
   * {
   *   QQ: "http://wpa.qq.com/msgrd?v=3&uin=1178522294&site=qq&menu=yes",
   *   Qzone: "https://1178522294.qzone.qq.com/",
   *   Gmail: "mailto:zhangbowang1998@gmail.com",
   *   Zhihu: "https://www.zhihu.com/people/mister-hope",
   *   Steam: "https://steamcommunity.com/id/Mr-Hope/",
   *   Weibo: "https://weibo.com/misterhope",
   * }
   * ```
   */
  links?: Partial<Record<BlogMedia, string>>;

  /**
   * 是否剪裁头像为圆形形状
   *
   * Whether cliping the avatar with round shape
   *
   * @default true
   */
  roundAvatar?: boolean;

  /**
   * 是否在侧边栏展示博主信息
   *
   * Whether to display blogger info in sidebar
   *
   * @default 'none'
   */
  sidebarDisplay?: "mobile" | "none" | "always";

  /**
   * 时间轴自定义文字
   *
   * Custom text for timeline
   *
   * @default 'Yesterday once more'
   */
  timeline?: string;
  /**
   * 每页的文章数量
   *
   * Article number per page
   *
   * @default 10
   */
  perPage?: number;
};

/**
 * 加密选项
 *
 * Encrypt Options
 */
export interface EncryptOptions {
  /**
   * 功能状态
   *
   * - `'global'` 意味着全局启用
   * - `'local'` 意味着全局禁用，可在页面内启用
   *
   * Feature Status
   *
   * - `'global'` means enabled globally
   * - `'local'` means disabled globally and can be enabled in pages
   *
   * @default 'local'
   */
  status?: "global" | "local";
  /**
   * 最高权限密码
   *
   * Global passwords, which has the highest authority
   */
  global?: string | string[];
  /**
   * 加密配置
   *
   * ```json
   * {
   *   // 这会加密整个 guide 目录，并且两个密码都是可用的
   *   "/guide/": ["1234", "5678"],
   *   // 这只会加密 config/page.html
   *   "/config/page.html": "1234"
   * }
   * ```
   *
   * Encrypt Configuration
   *
   * E.g.:
   *
   * ```json
   * {
   *   // This will encrypt the entire guide directory and both passwords will be available
   *   "/guide/": ["1234", "5678"],
   *   // this will only encrypt config/page.html
   *   "/config/page.html": "1234"
   * }
   * ```
   */
  config?: Record<string, string | string[]>;
}

/** 自定义布局配置 */
export interface CustomOptions {
  /** 页面顶部插槽 */
  pageTop?: string;
  /** 文章内容顶部插槽 */
  contentTop?: string;
  /** 文章内容底部插槽 */
  contentBottom?: string;
  /** 页面底部插槽 */
  pageBottom?: string;

  /** 导航栏起始插槽 */
  navbarStart?: string;
  /** 导航栏中部插槽 */
  navbarCenter?: string;
  /** 导航栏结束插槽 */
  navbarEnd?: string;

  /** 侧边栏顶部插槽 */
  sidebarTop?: string;
  /** 侧边栏中部插槽 */
  sidebarCenter?: string;
  /** 侧边栏底部插槽 */
  sidebarBottom?: string;
}

export interface HopeFeatureConfig {
  /**
   * 深色模式支持选项:
   *
   * - `'auto-switch'`: "关闭 | 自动 | 打开" 的三段式开关 (默认)
   * - `'switch'`: "关闭 | 打开" 的切换式开关
   * - `'auto'`: 自动根据用户设备主题或当前时间决定是否应用深色模式
   * - `'disable'`: 禁用深色模式
   *
   * Dark mode support options:
   *
   * - `'auto-switch'`: "off | automatic | on" three-stage switch (Default)
   * - `'switch'`: "Close | Open" toggle switch
   * - `'auto'`: Automatically decide whether to apply dark mode based on user device’s color-scheme or current time
   * - `'disable'`: disable dark mode
   *
   * @default 'auto-switch'
   */
  darkmode?: "auto-switch" | "auto" | "switch" | "disable";

  /**
   * 主题色选项配置。
   *
   * Theme color configuration.
   *
   * E.g.:
   * ```js
   * {
   *   blue: '#2196f3',
   *   red: '#f26d6d',
   *   green: '#3eaf7c',
   *   orange: '#fb9b5f'
   * }
   * ```
   *
   * @default { blue: '#2196f3', red: '#f26d6d', green: '#3eaf7c', orange: '#fb9b5f' }
   */
  themeColor?: Record<string, string> | false;

  /**
   * 博客设置
   *
   * Blog configuration
   */
  blog?: BlogOptions | false;

  /**
   * 加密设置
   *
   * Encrypt Configuration
   */
  encrypt?: EncryptOptions;

  /**
   * 自定义组件设置
   */
  custom?: CustomOptions;

  /**
   * 是否启用平滑滚动
   *
   * Enable smooth scrolling feature
   *
   * @default true
   */
  smoothScroll?: boolean;

  /**
   * 每分钟的阅读字数
   *
   * Reading speed of word per minute
   *
   * @default 300
   */
  wordPerminute?: number;
}
