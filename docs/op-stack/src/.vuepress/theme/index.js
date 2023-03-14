"use strict";
const alias_1 = require("./node/alias");
const config_1 = require("./node/config");
const plugins_1 = require("./node/plugins");
// Theme API.
const themeAPI = (themeConfig, ctx) => ({
    alias: (0, alias_1.getAlias)(themeConfig, ctx),
    plugins: (0, plugins_1.getPluginConfig)(themeConfig),
    additionalPages: [],
});
themeAPI.config = config_1.config;
// helper functions
themeAPI.themeConfig = (themeConfig) => themeConfig;
themeAPI.navbarConfig = (navbarConfig) => navbarConfig;
themeAPI.sidebarConfig = (sidebarConfig) => sidebarConfig;
module.exports = themeAPI;
//# sourceMappingURL=index.js.map
