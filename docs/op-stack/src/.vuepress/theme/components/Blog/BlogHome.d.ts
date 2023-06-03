import Vue from "vue";
/**
 * 项目配置
 *
 * Project Configuration
 */
export interface ProjectOptions {
    /**
     * 项目类型
     *
     * Type of project
     */
    type: "article" | "book" | "link" | "project";
    /**
     * 项目名称
     *
     * Project name
     */
    name: string;
    /**
     * 项目描述
     *
     * Project desription
     */
    desc?: string;
    /**
     * 项目封面，应为绝对路径
     *
     * Cover for the project, must be an absolute path
     */
    cover?: string;
    /**
     * 项目链接
     *
     * Link of the project
     */
    link: string;
}
declare const _default: import("vue/types/vue").ExtendedVue<Vue, unknown, unknown, unknown, Record<never, any>>;
export default _default;
