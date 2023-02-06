import Vue from "vue";
import Baidu from "@theme/icons/media/Baidu.vue";
import Bitbucket from "@theme/icons/media/Bitbucket.vue";
import Dingding from "@theme/icons/media/Dingding.vue";
import Discord from "@theme/icons/media/Discord.vue";
import Dribbble from "@theme/icons/media/Dribbble.vue";
import Email from "@theme/icons/media/Email.vue";
import Evernote from "@theme/icons/media/Evernote.vue";
import Facebook from "@theme/icons/media/Facebook.vue";
import Flipboard from "@theme/icons/media/Flipboard.vue";
import Gitee from "@theme/icons/media/Gitee.vue";
import Github from "@theme/icons/media/Github.vue";
import Gitlab from "@theme/icons/media/Gitlab.vue";
import Gmail from "@theme/icons/media/Gmail.vue";
import Instagram from "@theme/icons/media/Instagram.vue";
import Lines from "@theme/icons/media/Lines.vue";
import Linkedin from "@theme/icons/media/Linkedin.vue";
import Pinterest from "@theme/icons/media/Pinterest.vue";
import Pocket from "@theme/icons/media/Pocket.vue";
import QQ from "@theme/icons/media/QQ.vue";
import Qzone from "@theme/icons/media/Qzone.vue";
import Reddit from "@theme/icons/media/Reddit.vue";
import Rss from "@theme/icons/media/Rss.vue";
import Steam from "@theme/icons/media/Steam.vue";
import Twitter from "@theme/icons/media/Twitter.vue";
import Wechat from "@theme/icons/media/Wechat.vue";
import Weibo from "@theme/icons/media/Weibo.vue";
import Whatsapp from "@theme/icons/media/Whatsapp.vue";
import Youtube from "@theme/icons/media/Youtube.vue";
import Zhihu from "@theme/icons/media/Zhihu.vue";
const medias = [
    "Baidu",
    "Bitbucket",
    "Dingding",
    "Discord",
    "Dribbble",
    "Email",
    "Evernote",
    "Facebook",
    "Flipboard",
    "Gitee",
    "Github",
    "Gitlab",
    "Gmail",
    "Instagram",
    "Lines",
    "Linkedin",
    "Pinterest",
    "Pocket",
    "QQ",
    "Qzone",
    "Reddit",
    "Rss",
    "Steam",
    "Twitter",
    "Wechat",
    "Weibo",
    "Whatsapp",
    "Youtube",
    "Zhihu",
];
export default Vue.extend({
    name: "MediaLinks",
    components: {
        Baidu,
        Bitbucket,
        Dingding,
        Discord,
        Dribbble,
        Email,
        Evernote,
        Facebook,
        Flipboard,
        Gitee,
        Github,
        Gitlab,
        Gmail,
        Instagram,
        Lines,
        Linkedin,
        Pinterest,
        Pocket,
        QQ,
        Qzone,
        Reddit,
        Rss,
        Steam,
        Twitter,
        Wechat,
        Weibo,
        Whatsapp,
        Youtube,
        Zhihu,
    },
    computed: {
        mediaLink() {
            const { medialink } = this.$frontmatter;
            return medialink === false
                ? false
                : typeof medialink === "object"
                    ? medialink
                    : this.$themeConfig.blog
                        ? this.$themeConfig.blog.links || false
                        : false;
        },
        links() {
            if (this.mediaLink) {
                const links = [];
                for (const media in this.mediaLink)
                    if (medias.includes(media))
                        links.push({
                            icon: media,
                            url: this.mediaLink[media],
                        });
                return links;
            }
            return [];
        },
    },
});
//# sourceMappingURL=MediaLinks.js.map