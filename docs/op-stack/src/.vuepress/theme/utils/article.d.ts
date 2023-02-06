import type { PageComputed } from "@mr-hope/vuepress-types";
export declare const getDate: (date: string | number | Date) => (number | undefined)[];
export declare const compareDate: (dataA: Date | number | string | undefined, dataB: Date | number | string | undefined) => number;
export declare const filterArticle: (pages: PageComputed[], filterFunc?: ((page: PageComputed) => boolean) | undefined) => PageComputed[];
export declare const sortArticle: (pages: PageComputed[], compareKey?: "sticky" | "star" | undefined) => PageComputed[];
export declare const generatePagination: (pages: PageComputed[], perPage?: number) => PageComputed[][];
