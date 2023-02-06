import Vue from "vue";
export default Vue.extend({
    name: "MyTransition",
    props: {
        delay: { type: Number, default: 0 },
        duration: { type: Number, default: 0.25 },
        disable: { type: Boolean, default: false },
    },
    methods: {
        setStyle(items) {
            if (this.disable) {
                return;
            }

            items.style.transition = `transform ${this.duration}s ease-in-out ${this.delay}s, opacity ${this.duration}s ease-in-out ${this.delay}s`;
            items.style.transform = "translateY(-20px)";
            items.style.opacity = "0";
        },
        unsetStyle(items) {
            if (this.disable) {
                return;
            }

            items.style.transform = "translateY(0)";
            items.style.opacity = "1";
        },
    },
});
//# sourceMappingURL=MyTransition.js.map