"use strict";
const alias_1 = require("./node/alias");
const config_1 = require("./node/config");
const eject_1 = require("./node/eject");
const plugins_1 = require("./node/plugins");
const blogAddtionalPages = [
    {
        path: "/article/",
        frontmatter: { layout: "Blog" },
    },
    {
        path: "/star/",
        frontmatter: { layout: "Blog" },
    },
    {
        path: "/encrypt/",
        frontmatter: { layout: "Blog" },
    },
    {
        path: "/slide/",
        frontmatter: { layout: "Blog" },
    },
    {
        path: "/timeline/",
        frontmatter: { layout: "Blog" },
    },
];
// Theme API.
const themeAPI = (themeConfig, ctx) => ({
    alias: (0, alias_1.getAlias)(themeConfig, ctx),
    plugins: (0, plugins_1.getPluginConfig)(themeConfig),
    additionalPages: themeConfig.blog === false ? [] : blogAddtionalPages,
    extendCli: (cli) => {
        cli
            .command("eject-hope [targetDir]", "copy vuepress-theme-hope into .vuepress/theme for customization.")
            .option("--debug", "eject in debug mode")
            .action((dir) => {
            void (0, eject_1.eject)(dir || ".");
        });
    },
});
themeAPI.config = config_1.config;
// helper functions
themeAPI.themeConfig = (themeConfig) => themeConfig;
themeAPI.navbarConfig = (navbarConfig) => navbarConfig;
themeAPI.sidebarConfig = (sidebarConfig) => sidebarConfig;
module.exports = themeAPI;
//# sourceMappingURL=index.js.map