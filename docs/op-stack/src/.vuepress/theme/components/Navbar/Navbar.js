import Vue from "vue";
import AlgoliaSearchBox from "@AlgoliaSearchBox";
import LanguageDropdown from "@theme/components/Navbar/LanguageDropdown";
import NavLinks from "@theme/components/Navbar/NavLinks.vue";
import RepoLink from "@theme/components/Navbar/RepoLink.vue";
import SearchBox from "@SearchBox";
import SidebarButton from "@theme/components/Navbar/SidebarButton.vue";
import ThemeColor from "@ThemeColor";
let handler;
const css = (el, property) => {
    // NOTE: Known bug, will return 'auto' if style value is 'auto'
    const window = el.ownerDocument.defaultView;
    // `null` means not to return pseudo styles
    // eslint-disable-next-line @typescript-eslint/no-non-null-assertion
    return window.getComputedStyle(el, null)[property];
};
export default Vue.extend({
    name: "Navbar",
    components: {
        AlgoliaSearchBox,
        LanguageDropdown,
        NavLinks,
        RepoLink,
        SearchBox,
        SidebarButton,
        ThemeColor,
    },
    data: () => ({
        linksWrapMaxWidth: 0,
        isMobile: false,
    }),
    computed: {
        siteBrandTitle() {
            return this.$site.title;
        },
        canHideSiteBrandTitle() {
            return (Boolean(this.siteBrandTitle) &&
                this.$themeConfig.hideSiteTitleonMobile !== false);
        },
        siteBrandLogo() {
            const { logo } = this.$themeConfig;
            return logo ? this.$withBase(logo) : "";
        },
        siteBrandDarkLogo() {
            const { darkLogo } = this.$themeConfig;
            return darkLogo ? this.$withBase(darkLogo) : "";
        },
        algoliaConfig() {
            return (this.$themeLocaleConfig.algolia || this.$themeConfig.algolia || false);
        },
        isAlgoliaSearch() {
            return Boolean(this.algoliaConfig &&
                this.algoliaConfig.apiKey &&
                this.algoliaConfig.indexName);
        },
        canHide() {
            const autoHide = this.$themeConfig.navAutoHide;
            return autoHide !== "none" && (autoHide === "always" || this.isMobile);
        },
    },
    mounted() {
        // Refer to config.styl
        const MOBILE_DESKTOP_BREAKPOINT = 719;
        const NAVBAR_HORIZONTAL_PADDING = parseInt(css(this.$el, "paddingLeft")) +
            parseInt(css(this.$el, "paddingRight"));
        handler = () => {
            if (document.documentElement.clientWidth < MOBILE_DESKTOP_BREAKPOINT) {
                this.isMobile = true;
                this.linksWrapMaxWidth = 0;
            }
            else {
                this.isMobile = false;
                this.linksWrapMaxWidth =
                    this.$el.offsetWidth -
                        NAVBAR_HORIZONTAL_PADDING -
                        ((this.$refs.siteInfo &&
                            this.$refs.siteInfo.$el &&
                            this.$refs.siteInfo.$el.offsetWidth) ||
                            0);
            }
        };
        handler();
        window.addEventListener("resize", handler);
        window.addEventListener("orientationchange", handler);
    },
    // eslint-disable-next-line vue/no-deprecated-destroyed-lifecycle
    beforeDestroy() {
        window.removeEventListener("resize", handler);
        window.removeEventListener("orientationchange", handler);
    },
});
//# sourceMappingURL=Navbar.js.map