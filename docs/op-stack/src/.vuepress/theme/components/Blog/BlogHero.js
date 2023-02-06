import Vue from "vue";
import MyTransition from "@theme/components/MyTransition.vue";
import defaultHeroImage from "@theme/assets/hero.jpg";
export default Vue.extend({
    name: "BlogHero",
    components: { MyTransition },
    data: () => ({ defaultHeroImage }),
    computed: {
        heroImageStyle() {
            const defaultStyle = {
                maxHeight: "180px",
                margin: this.$frontmatter.showTitle === false
                    ? "6rem auto 1.5rem"
                    : "1rem auto",
            };
            return Object.assign(Object.assign({}, defaultStyle), this.$frontmatter.heroImageStyle);
        },
        bgImageStyle() {
            const defaultBgImageStyle = {
                height: "350px",
                textAlign: "center",
                overflow: "hidden",
            };
            const { bgImageStyle = {} } = this.$frontmatter;
            return Object.assign(Object.assign({}, defaultBgImageStyle), bgImageStyle);
        },
    },
});
//# sourceMappingURL=BlogHero.js.map