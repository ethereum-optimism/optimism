import Vue from "vue";
export default Vue.extend({
    name: "Clipboard",
    props: {
        html: { type: String, default: "" },
        lang: { type: String, default: "en-US" },
    },
    data: () => ({
        location: "",
    }),
    computed: {
        copyright() {
            const { author } = this.$themeConfig;
            const content = {
                "zh-CN": `${this.html}\n-----\n${author ? `著作权归${author}所有。\n` : ""}链接: ${this.location}`,
                "en-US": `${this.html}\n-----\n${author ? `Copyright by ${author}.\n` : ""}Link: ${this.location}`,
                "vi-VN": `${this.html}\n-----\n${author ? `bản quyền bởi ${author}.\n` : ""}Liên kết: ${this.location}`,
            };
            return content[this.lang];
        },
    },
    created() {
        if (typeof window !== "undefined")
            this.location = window.location.toString();
    },
});
//# sourceMappingURL=Clipboard.js.map