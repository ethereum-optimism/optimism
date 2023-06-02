"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getAlias = void 0;
const path_1 = require("path");
const getAlias = (themeConfig, ctx) => {
    const { siteConfig } = ctx;
    // Resolve algolia
    const isAlgoliaSearch = Boolean(themeConfig.algolia) ||
        Object.keys((siteConfig.locales && themeConfig.locales) || {}).some((base) => themeConfig.locales[base].algolia);
    const blogEnabled = themeConfig.blog !== false;
    const commentEnabled = themeConfig.comment &&
        themeConfig.comment.type &&
        themeConfig.comment.type !== "disable";
    const themeColorEnabled = !(themeConfig.themeColor === false && themeConfig.darkmode === "disable");
    const { custom = {} } = themeConfig;
    const noopModule = "@mr-hope/vuepress-shared/lib/esm/noopModule";
    return {
        "@AlgoliaSearchBox": isAlgoliaSearch
            ? themeConfig.algoliaType === "full"
                ? (0, path_1.resolve)(__dirname, "../components/AlgoliaSearch/Full.vue")
                : (0, path_1.resolve)(__dirname, "../components/AlgoliaSearch/Dropdown.vue")
            : noopModule,
        "@BlogInfo": blogEnabled
            ? (0, path_1.resolve)(__dirname, "../components/Blog/BlogInfo.vue")
            : noopModule,
        "@BloggerInfo": blogEnabled
            ? (0, path_1.resolve)(__dirname, "../components/Blog/BloggerInfo.vue")
            : noopModule,
        "@BlogHome": blogEnabled
            ? (0, path_1.resolve)(__dirname, "../components/Blog/BlogHome.vue")
            : noopModule,
        "@BlogPage": blogEnabled
            ? (0, path_1.resolve)(__dirname, "../components/Blog/BlogPage.vue")
            : noopModule,
        "@ContentTop": custom.contentTop
            ? (0, path_1.resolve)(ctx.sourceDir, ".vuepress", custom.contentTop)
            : noopModule,
        "@ContentBottom": custom.contentBottom
            ? (0, path_1.resolve)(ctx.sourceDir, ".vuepress", custom.contentBottom)
            : noopModule,
        "@PageTop": custom.pageTop
            ? (0, path_1.resolve)(ctx.sourceDir, ".vuepress", custom.pageTop)
            : noopModule,
        "@PageBottom": custom.pageBottom
            ? (0, path_1.resolve)(ctx.sourceDir, ".vuepress", custom.pageBottom)
            : noopModule,
        "@Comment": commentEnabled
            ? "@mr-hope/vuepress-plugin-comment/lib/client/Comment.vue"
            : noopModule,
        "@NavbarStart": custom.navbarStart
            ? (0, path_1.resolve)(ctx.sourceDir, ".vuepress", custom.navbarStart)
            : noopModule,
        "@NavbarCenter": custom.navbarCenter
            ? (0, path_1.resolve)(ctx.sourceDir, ".vuepress", custom.navbarCenter)
            : noopModule,
        "@NavbarEnd": custom.navbarEnd
            ? (0, path_1.resolve)(ctx.sourceDir, ".vuepress", custom.navbarEnd)
            : noopModule,
        "@ThemeColor": themeColorEnabled
            ? (0, path_1.resolve)(__dirname, "../components/Theme/ThemeColor.vue")
            : noopModule,
        "@SidebarTop": custom.sidebarTop
            ? (0, path_1.resolve)(ctx.sourceDir, ".vuepress", custom.sidebarTop)
            : noopModule,
        "@SidebarCenter": custom.sidebarCenter
            ? (0, path_1.resolve)(ctx.sourceDir, ".vuepress", custom.sidebarCenter)
            : noopModule,
        "@SidebarBottom": custom.sidebarBottom
            ? (0, path_1.resolve)(ctx.sourceDir, ".vuepress", custom.sidebarBottom)
            : noopModule,
    };
};
exports.getAlias = getAlias;
//# sourceMappingURL=alias.js.map