import type { DirectiveOptions, VNode } from "vue";
import type { DirectiveBinding } from "vue/types/options";
declare type Event = TouchEvent | MouseEvent;
interface PopupHtmlElement extends HTMLElement {
    $vueClickOutside?: {
        callback: (event: Event) => void;
        handler: (event: Event) => void;
    };
}
declare type PopupDirectiveFunction = (el: PopupHtmlElement, binding: DirectiveBinding, vnode: VNode, oldVnode: VNode) => void;
export declare const bind: PopupDirectiveFunction;
export declare const update: PopupDirectiveFunction;
export declare const unbind: PopupDirectiveFunction;
declare const _default: DirectiveOptions;
export default _default;
