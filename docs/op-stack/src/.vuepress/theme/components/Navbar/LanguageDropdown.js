import Vue from "vue";
import DropdownLink from "@theme/components/Navbar/DropdownLink.vue";
import I18nIcon from "@theme/icons/I18nIcon.vue";
import NavLink from "@theme/components/Navbar/NavLink.vue";
import { getNavLinkItem } from "@theme/utils/navbar";
export default Vue.extend({
    name: "LanguageDropdown",
    components: { NavLink, DropdownLink },
    computed: {
        dropdown() {
            const { locales } = this.$site;
            if (locales && Object.keys(locales).length > 1) {
                const currentLink = this.$page.path;
                const { routes } = this.$router.options;
                const themeLocales = this.$themeConfig.locales || {};
                const languageDropdown = {
                    text: this.$themeLocaleConfig.selectText || "Languages",
                    ariaLabel: this.$themeLocaleConfig.ariaLabel || "Select language",
                    items: Object.keys(locales).map((path) => {
                        const locale = locales[path];
                        const text = (themeLocales[path] && themeLocales[path].label) ||
                            locale.lang ||
                            "Unknown Language";
                        let link;
                        // Stay on the current page
                        if (locale.lang === this.$lang)
                            link = currentLink;
                        else {
                            // Try to stay on the same page
                            link = currentLink.replace(this.$localeConfig.path, path);
                            // Fallback to homepage
                            if (!(routes || []).some((route) => route.path === link))
                                link = path;
                        }
                        return { text, link };
                    }),
                };
                return getNavLinkItem(languageDropdown);
            }
            return false;
        },
    },
    render(h) {
        return this.dropdown
            ? h("div", { class: "nav-links" }, [
                h("div", { class: "nav-item" }, [
                    h(DropdownLink, { props: { item: this.dropdown } }, [
                        h(I18nIcon, {
                            slot: "title",
                            style: {
                                width: "1rem",
                                height: "1rem",
                                verticalAlign: "middle",
                                marginLeft: "1rem",
                            },
                        }),
                    ]),
                ]),
            ])
            : null;
    },
});
//# sourceMappingURL=LanguageDropdown.js.map