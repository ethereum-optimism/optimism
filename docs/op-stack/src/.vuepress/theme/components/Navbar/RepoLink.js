import Vue from "vue";
export default Vue.extend({
    name: "RepoLink",
    computed: {
        repoLink() {
            const { repo } = this.$themeConfig;
            if (repo)
                return /^https?:/u.test(repo) ? repo : `https://github.com/${repo}`;
            return "";
        },
        repoLabel() {
            if (!this.repoLink)
                return "";
            if (this.$themeConfig.repoLabel)
                return this.$themeConfig.repoLabel;
            const [repoHost] = /^https?:\/\/[^/]+/u.exec(this.repoLink) || [""];
            const platforms = ["GitHub", "GitLab", "Bitbucket"];
            for (let index = 0; index < platforms.length; index++) {
                const platform = platforms[index];
                if (new RegExp(platform, "iu").test(repoHost))
                    return platform;
            }
            return "Source";
        },
    },
});
//# sourceMappingURL=RepoLink.js.map