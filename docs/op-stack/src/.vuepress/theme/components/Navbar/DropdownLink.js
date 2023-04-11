import Vue from "vue";
import NavLink from "@theme/components/Navbar/NavLink.vue";
export default Vue.extend({
    name: "DropdownLink",
    components: { NavLink },
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
        handleDropdown(event) {
            const isTriggerByTab = event.detail === 0;
            if (isTriggerByTab)
                this.setOpen(!this.open);
        },
        isLastItemOfArray(item, array) {
            if (Array.isArray(array))
                return item === array[array.length - 1];
            return false;
        },
    },
});
//# sourceMappingURL=DropdownLink.js.map