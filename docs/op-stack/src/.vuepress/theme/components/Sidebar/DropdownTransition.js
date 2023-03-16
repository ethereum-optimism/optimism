import Vue from "vue";
export default Vue.extend({
    name: "DropdownTransition",
    methods: {
        setHeight(items) {
            // explicitly set height so that it can be transitioned
            items.style.height = `${items.scrollHeight}px`;
        },
        unsetHeight(items) {
            items.style.height = "";
        },
    },
});
//# sourceMappingURL=DropdownTransition.js.map