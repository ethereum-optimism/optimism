import Vue from "vue";
import DropdownTransition from "@theme/components/Sidebar/DropdownTransition.vue";
import NavLink from "@theme/components/Navbar/NavLink.vue";
export default Vue.extend({
    name: "SidebarDropdownLink",
    components: { NavLink, DropdownTransition },
    props: {
        item: { type: Object, required: true },
    },
    data: () => ({
        open: false,
    }),
    computed: {
        dropdownAriaLabel() {
            return this.item.ariaLabel || this.item.text;
        },
        iconPrefix() {
            const { iconPrefix } = this.$themeConfig;
            return iconPrefix === "" ? "" : iconPrefix || "icon-";
        },
    },
    watch: {
        $route() {
            this.open = false;
        },
    },
    methods: {
        setOpen(value) {
            this.open = value;
        },
        isLastItemOfArray(item, array) {
            if (Array.isArray(array))
                return item === array[array.length - 1];
            return false;
        },
    },
});
//# sourceMappingURL=SidebarDropdownLink.js.map