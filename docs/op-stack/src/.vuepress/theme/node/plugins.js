"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getPluginConfig = void 0;
const path_1 = require("path");
const clean_url_1 = require("./clean-url");
const chunk_rename_1 = require("./chunk-rename");
const getPluginConfig = (themeConfig) => {
    // set author for comment plugin
    if (themeConfig.comment && themeConfig.author)
        themeConfig.comment.author = themeConfig.author;
    return [
        ["@mr-hope/comment", themeConfig.comment || true],
        ["@mr-hope/components"],
        ["@mr-hope/feed", themeConfig.feed],
        ["@mr-hope/git", themeConfig.git],
        ["@mr-hope/pwa", themeConfig.pwa],
        ["@mr-hope/seo", themeConfig.seo],
        ["@mr-hope/sitemap", themeConfig.sitemap],
        [
            "@mr-hope/smooth-scroll",
            themeConfig.smoothScroll === false
                ? false
                : typeof themeConfig.smoothScroll === "number"
                    ? { delay: themeConfig.smoothScroll }
                    : themeConfig.smoothScroll || { delay: 500 },
        ],
        [
            "@vuepress/blog",
            themeConfig.blog === false
                ? false
                : {
                    frontmatters: [
                        {
                            id: "tag",
                            keys: ["tag", "tags"],
                            path: "/tag/",
                            layout: "Blog",
                            scopeLayout: "Blog",
                        },
                        {
                            id: "category",
                            keys: ["category", "categories"],
                            path: "/category/",
                            layout: "Blog",
                            scopeLayout: "Blog",
                        },
                    ],
                },
        ],
        ["@vuepress/last-updated", false],
        "@vuepress/nprogress",
        [
            "@vuepress/search",
            {
                searchMaxSuggestions: themeConfig.searchMaxSuggestions || 10,
            },
        ],
        ["active-hash", themeConfig.activeHash],
        ["add-this", typeof themeConfig.addThis === "string"],
        [
            "copyright",
            typeof themeConfig.copyright === "object"
                ? Object.assign({ minLength: 100, disable: themeConfig.copyright.status === "local", clipboardComponent: (0, path_1.resolve)(__dirname, "../components/Clipboard.vue") }, themeConfig.copyright) : false,
        ],
        ["md-enhance", themeConfig.mdEnhance || {}],
        ["@mr-hope/copy-code", themeConfig.copyCode],
        ["photo-swipe", themeConfig.photoSwipe],
        [
            "typescript",
            themeConfig.typescript
                ? {
                    tsLoaderOptions: typeof themeConfig.typescript === "object"
                        ? themeConfig.typescript
                        : {},
                }
                : false,
        ],
        [
            clean_url_1.cleanUrlPlugin,
            themeConfig.cleanUrl === false
                ? false
                : themeConfig.cleanUrl || { normalSuffix: "/" },
        ],
        [
            chunk_rename_1.chunkRenamePlugin,
            themeConfig.chunkRename === false ? false : themeConfig.chunkRename,
        ],
    ];
};
exports.getPluginConfig = getPluginConfig;
//# sourceMappingURL=plugins.js.map