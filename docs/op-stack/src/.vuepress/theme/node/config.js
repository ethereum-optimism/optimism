"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.config = void 0;
const vuepress_shared_1 = require("@mr-hope/vuepress-shared");
const locales_1 = require("./locales");
const themeConfig_1 = require("./themeConfig");
const defaultConfig = {
    base: process.env.VuePress_BASE || "/",
    temp: "./node_modules/.temp",
    theme: "hope",
    themeConfig: { locales: {} },
    evergreen: true,
};
const getRootLang = (config) => {
    var _a, _b;
    // infer from siteLocale
    const siteLocales = config.locales;
    if ((siteLocales === null || siteLocales === void 0 ? void 0 : siteLocales["/"]) && (0, vuepress_shared_1.checkLang)((_a = siteLocales["/"]) === null || _a === void 0 ? void 0 : _a.lang))
        return siteLocales["/"].lang;
    // infer from themeLocale
    const themeLocales = config.locales;
    if ((themeLocales === null || themeLocales === void 0 ? void 0 : themeLocales["/"]) && (0, vuepress_shared_1.checkLang)((_b = themeLocales["/"]) === null || _b === void 0 ? void 0 : _b.lang))
        return themeLocales["/"].lang;
    (0, vuepress_shared_1.showLangError)("root");
    return "en-US";
};
const config = (config) => {
    // merge default config
    (0, vuepress_shared_1.deepAssignReverse)(defaultConfig, config);
    const resolvedConfig = config;
    const rootLang = getRootLang(resolvedConfig);
    (0, themeConfig_1.resolveThemeConfig)(resolvedConfig.themeConfig, rootLang);
    (0, locales_1.resolveLocales)(resolvedConfig, rootLang);
    return resolvedConfig;
};
exports.config = config;
//# sourceMappingURL=config.js.map