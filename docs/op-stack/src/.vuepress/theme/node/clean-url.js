"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.cleanUrlPlugin = void 0;
const cleanUrlPlugin = ({ normalSuffix = "", indexSuffix = "/", notFoundPath = "/404.html", }) => ({
    name: "clean-url",
    extendPageData(page) {
        const { regularPath, frontmatter = {} } = page;
        if (!frontmatter.permalink) {
            if (regularPath === "/404.html")
                // path for 404 page
                page.path = notFoundPath;
            else if (regularPath.endsWith(".html"))
                // normal path
                // e.g. foo/bar.md -> foo/bar.html
                page.path = `${regularPath.slice(0, -5)}${normalSuffix}`;
            else if (regularPath.endsWith("/"))
                // index path
                // e.g. foo/index.md -> foo/
                page.path = `${regularPath.slice(0, -1)}${indexSuffix}`;
        }
    },
});
exports.cleanUrlPlugin = cleanUrlPlugin;
//# sourceMappingURL=clean-url.js.map