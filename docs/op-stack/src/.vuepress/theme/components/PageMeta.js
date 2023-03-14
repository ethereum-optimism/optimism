import Vue from "vue";
import EditIcon from "@theme/icons/EditIcon.vue";
import { endingSlashRE, outboundRE } from "@theme/utils/path";
export default Vue.extend({
    name: "PageMeta",
    components: { EditIcon },
    computed: {
        i18n() {
            return (this.$themeLocaleConfig.meta || {
                contributor: "Contributors",
                editLink: "Edit this page",
                updateTime: "Last Updated",
            });
        },
        contributors() {
            return this.$page.frontmatter.contributor === false ||
                (this.$themeConfig.contributor === false &&
                    !this.$page.frontmatter.contributor)
                ? []
                : this.$page.contributors || [];
        },
        contributorsText() {
            return this.i18n.contributor;
        },
        updateTime() {
            return this.$page.frontmatter.contributor === false ||
                (this.$themeConfig.updateTime === false &&
                    !this.$page.frontmatter.updateTime)
                ? ""
                : this.$page.updateTime || "";
        },
        updateTimeText() {
            return this.i18n.updateTime;
        },
        editLink() {
            const showEditLink = this.$page.frontmatter.editLink ||
                (this.$themeConfig.editLinks !== false &&
                    this.$page.frontmatter.editLink !== false);
            const { repo, docsRepo } = this.$site.themeConfig;
            if (showEditLink && (repo || docsRepo) && this.$page.relativePath)
                return this.createEditLink();
            return false;
        },
        editLinkText() {
            return this.i18n.editLink;
        },
    },
    methods: {
        createEditLink() {
            const { repo = "", docsRepo = repo, docsDir = "", docsBranch = "main", } = this.$themeConfig;
            const bitbucket = /bitbucket.org/u;
            if (bitbucket.test(docsRepo))
                return `${docsRepo.replace(endingSlashRE, "")}/src/${docsBranch}/${docsDir ? `${docsDir.replace(endingSlashRE, "")}/` : ""}${this.$page.relativePath}?mode=edit&spa=0&at=${docsBranch}&fileviewer=file-view-default`;
            const gitlab = /gitlab.com/u;
            if (gitlab.test(docsRepo))
                return `${docsRepo.replace(endingSlashRE, "")}/-/edit/${docsBranch}/${docsDir ? `${docsDir.replace(endingSlashRE, "")}/` : ""}${this.$page.relativePath}`;
            const base = outboundRE.test(docsRepo)
                ? docsRepo
                : `https://github.com/${docsRepo}`;
            return `${base.replace(endingSlashRE, "")}/edit/${docsBranch}/${docsDir ? `${docsDir.replace(endingSlashRE, "")}/` : ""}${this.$page.relativePath}`;
        },
    },
});
//# sourceMappingURL=PageMeta.js.map