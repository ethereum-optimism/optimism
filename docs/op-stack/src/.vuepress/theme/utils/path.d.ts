import type { Route } from "vue-router";
export declare const hashRE: RegExp;
export declare const extRE: RegExp;
export declare const endingSlashRE: RegExp;
export declare const outboundRE: RegExp;
/** Remove hash and ext in a link */
export declare const normalize: (path: string) => string;
export declare const getHash: (path: string) => string | void;
/** Judge whether a path is external */
export declare const isExternal: (path: string) => boolean;
/** Judge whether a path is `mailto:` link */
export declare const isMailto: (path: string) => boolean;
/** Judge whether a path is `tel:` link */
export declare const isTel: (path: string) => boolean;
export declare const ensureExt: (path: string) => string;
export declare const ensureEndingSlash: (path: string) => string;
/** Judge whether a route match a link */
export declare const isActive: (route: Route, path: string) => boolean;
/**
 * @param path links being resolved
 * @param base deploy base
 * @param append whether append directly
 */
export declare const resolvePath: (path: string, base: string, append?: boolean | undefined) => string;
