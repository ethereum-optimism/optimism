import { compareSync } from "bcryptjs";
import { encryptBaseMixin } from "@theme/mixins/encrypt";
export const globalEncryptMixin = encryptBaseMixin.extend({
    data: () => ({
        globalEncryptPassword: "",
    }),
    computed: {
        isGlobalEncrypted() {
            if (this.encryptOptions.status === "global" &&
                this.encryptOptions.global) {
                const { global } = this.encryptOptions;
                const globalPasswords = typeof global === "string" ? [global] : global;
                // none of the password matches
                return !globalPasswords.some((globalPassword) => compareSync(this.globalEncryptPassword, globalPassword));
            }
            return false;
        },
    },
    mounted() {
        const globalPassword = localStorage.getItem("globalPassword");
        if (globalPassword)
            this.globalEncryptPassword = globalPassword;
    },
    methods: {
        checkGlobalPassword(globalPassword) {
            const { global } = this.encryptOptions;
            const globalPasswords = typeof global === "string" ? [global] : global;
            if (
            // some of the password matches
            globalPasswords.some((password) => compareSync(globalPassword, password))) {
                this.globalEncryptPassword = globalPassword;
                localStorage.setItem("globalPassword", globalPassword);
            }
        },
    },
});
//# sourceMappingURL=globalEncrypt.js.map