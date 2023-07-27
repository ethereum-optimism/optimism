import BlogInfo from "@BlogInfo";
import BlogPage from "@BlogPage";
import Common from "@theme/components/Common.vue";
import MyTransition from "@theme/components/MyTransition.vue";
import { globalEncryptMixin } from "@theme/mixins/globalEncrypt";
import { pathEncryptMixin } from "@theme/mixins/pathEncrypt";
import Password from "@theme/components/Password.vue";
export default globalEncryptMixin.extend(pathEncryptMixin).extend({
    components: {
        BlogInfo,
        BlogPage,
        Common,
        MyTransition,
        Password,
    },
});
//# sourceMappingURL=Blog.js.map