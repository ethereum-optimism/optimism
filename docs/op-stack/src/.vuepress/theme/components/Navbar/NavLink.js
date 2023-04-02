import Vue from "vue";
import { ensureExt, isExternal, isMailto, isTel } from "@theme/utils/path";
export default Vue.extend({
    name: "NavLink",
    props: {
        item: { type: Object, required: true },
    },
    computed: {
        link() {
            return ensureExt(this.item.link);
        },
        iconPrefix() {
            const { iconPrefix } = this.$themeConfig;
            return iconPrefix === "" ? "" : iconPrefix || "icon-";
        },
        active() {
            // link is home path
            if ((this.$site.locales &&
                Object.keys(this.$site.locales).some((rootLink) => rootLink === this.link)) ||
                this.link === "/")
                // exact match
                return this.$route.path === this.link;
            // inclusive match
            return this.$route.path.startsWith(this.link);
        },
        isNonHttpURI() {
            return isMailto(this.link) || isTel(this.link);
        },
        isBlankTarget() {
            return this.target === "_blank";
        },
        isInternal() {
            return !isExternal(this.link) && !this.isBlankTarget;
        },
        target() {
            if (this.isNonHttpURI)
                return null;
            if (this.item.target)
                return this.item.target;
            return isExternal(this.link) ? "_blank" : "";
        },
        rel() {
            if (this.isNonHttpURI)
                return null;
            if (this.item.rel === false)
                return null;
            if (this.item.rel)
                return this.item.rel;
            return this.isBlankTarget ? "noopener noreferrer" : null;
        },
    },
    methods: {
        focusoutAction() {
            // eslint-disable-next-line vue/require-explicit-emits
            this.$emit("focusout");
        },
    },
});
//# sourceMappingURL=NavLink.js.map