"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.eject = void 0;
const chalk_1 = require("chalk");
const fs_extra_1 = require("fs-extra");
const path_1 = require("path");
// #region exclude-files
const EXCLUDED_FILES = [
    "__tests__",
    ".npmignore",
    "LICENSE",
    "package.json",
    "node_modules",
    "README.md",
    "readme.md",
];
// #endregion exclude-files
const eject = async (dir) => {
    try {
        const sourceDir = (0, path_1.resolve)(__dirname, "../");
        const targetDir = (0, path_1.resolve)(process.cwd(), dir, ".vuepress/theme");
        await (0, fs_extra_1.copy)(sourceDir, targetDir, {
            filter: (src) => {
                return !EXCLUDED_FILES.includes((0, path_1.relative)(sourceDir, src));
            },
        });
        console.log(`Copied vuepress-theme-hope into ${(0, chalk_1.cyan)(targetDir)}.\n`);
    }
    catch (err) {
        console.error((0, chalk_1.red)(err.stack || ""));
        process.exitCode = 1;
    }
};
exports.eject = eject;
//# sourceMappingURL=eject.js.map