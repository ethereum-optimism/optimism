import { compareSync } from "bcryptjs";
import { encryptBaseMixin } from "@theme/mixins/encrypt";
import { getPathMatchedKeys } from "@theme/utils/encrypt";
export const pathEncryptMixin = encryptBaseMixin.extend({
    data: () => ({
        encryptPasswordConfig: {},
    }),
    computed: {
        pathEncryptMatchKeys() {
            return getPathMatchedKeys(this.encryptOptions, this.$route.path);
        },
        isPathEncrypted() {
            if (this.pathEncryptMatchKeys.length === 0)
                return false;
            const { config } = this.encryptOptions;
            // none of the password matches
            return this.pathEncryptMatchKeys.every((key) => {
                const keyConfig = config[key];
                const hitPasswords = typeof keyConfig === "string" ? [keyConfig] : keyConfig;
                return (!this.encryptPasswordConfig[key] ||
                    hitPasswords.every((encryptPassword) => !compareSync(this.encryptPasswordConfig[key], encryptPassword)));
            });
        },
    },
    mounted() {
        const passwordConfig = localStorage.getItem("encryptConfig");
        if (passwordConfig)
            this.encryptPasswordConfig = JSON.parse(passwordConfig);
    },
    methods: {
        checkPathPassword(password) {
            const { config } = this.$themeConfig.encrypt;
            for (const hitKey of this.pathEncryptMatchKeys) {
                const hitPassword = config[hitKey];
                const hitPasswordList = typeof hitPassword === "string" ? [hitPassword] : hitPassword;
                // some of the password matches
                if (hitPasswordList.filter((encryptPassword) => compareSync(password, encryptPassword))) {
                    this.$set(this.encryptPasswordConfig, hitKey, password);
                    localStorage.setItem("encryptConfig", JSON.stringify(this.encryptPasswordConfig));
                    break;
                }
            }
        },
    },
});
//# sourceMappingURL=pathEncrypt.js.map