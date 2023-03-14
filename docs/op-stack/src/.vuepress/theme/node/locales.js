"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.resolveLocales = void 0;
const vuepress_shared_1 = require("@mr-hope/vuepress-shared");
const resolveLocales = (config, rootLang) => {
    // ensure locales config
    if (!config.locales)
        config.locales = {};
    const { locales } = config;
    // set locate for base
    locales["/"] = Object.assign({ lang: rootLang }, (locales["/"] || {}));
    // handle other languages
    Object.keys(config.themeConfig.locales).forEach((path) => {
        if (path === "/")
            return;
        locales[path] = Object.assign({ lang: (0, vuepress_shared_1.path2Lang)(path) }, (locales[path] || {}));
    });
};
exports.resolveLocales = resolveLocales;
//# sourceMappingURL=locales.js.map