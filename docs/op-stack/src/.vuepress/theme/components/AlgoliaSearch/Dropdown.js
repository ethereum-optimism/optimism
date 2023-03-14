import Vue from "vue";
export default Vue.extend({
    name: "AlgoliaSearchDropdown",
    props: {
        options: { type: Object, required: true },
    },
    data: () => ({
        placeholder: "",
    }),
    watch: {
        $lang(newValue) {
            this.update(this.options, newValue);
        },
        options(newValue) {
            this.update(newValue, this.$lang);
        },
    },
    mounted() {
        this.initialize(this.options, this.$lang);
        this.placeholder =
            this.$site.themeConfig.searchPlaceholder || "";
    },
    methods: {
        initialize(userOptions, lang) {
            void Promise.all([
                import(
                /* webpackChunkName: "docsearch" */ "docsearch.js/dist/cdn/docsearch.min.js"),
                import(
                /* webpackChunkName: "docsearch" */ "docsearch.js/dist/cdn/docsearch.min.css"),
            ]).then(([docsearch]) => {
                // eslint-disable-next-line
                docsearch.default(Object.assign(Object.assign({}, userOptions), { inputSelector: "#algolia-search-input", 
                    // #697 Make docsearch work well at i18n mode.
                    algoliaOptions: {
                        facetFilters: [`lang:${lang}`].concat(
                        // eslint-disable-next-line
                        userOptions.facetFilters || []),
                    }, handleSelected: (_input, _event, suggestion) => {
                        const { pathname, hash } = new URL(suggestion.url);
                        const routepath = pathname.replace(this.$site.base, "/");
                        if (this.$router.getRoutes().some((route) => route.path === routepath))
                            void this.$router.push(`${routepath}${decodeURIComponent(hash)}`);
                        else
                            window.open(suggestion.url);
                    } }));
            });
        },
        update(options, lang) {
            this.$el.innerHTML =
                '<input id="algolia-search-input" class="search-query">';
            this.initialize(options, lang);
        },
    },
});
//# sourceMappingURL=Dropdown.js.map