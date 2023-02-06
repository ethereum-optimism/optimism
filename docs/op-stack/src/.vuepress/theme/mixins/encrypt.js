import Vue from "vue";
export const encryptBaseMixin = Vue.extend({
    computed: {
        encryptOptions() {
            return this.$themeConfig.encrypt || {};
        },
    },
});
//# sourceMappingURL=encrypt.js.map