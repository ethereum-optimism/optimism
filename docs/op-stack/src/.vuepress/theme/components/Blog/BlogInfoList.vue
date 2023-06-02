<template>
  <div class="blog-info-list">
    <div class="switch-wrapper">
      <button class="switch-button" @click="setActive('article')">
        <div
          class="icon-wapper"
          :class="{ active: active === 'article' }"
          :aria-label="i18n.article"
          data-balloon-pos="up"
        >
          <ArticleIcon />
        </div>
      </button>
      <button class="switch-button" @click="setActive('category')">
        <div
          class="icon-wapper"
          :class="{ active: active === 'category' }"
          :aria-label="i18n.category"
          data-balloon-pos="up"
        >
          <CategoryIcon />
        </div>
      </button>
      <button class="switch-button" @click="setActive('tag')">
        <div
          class="icon-wapper"
          :class="{ active: active === 'tag' }"
          :aria-label="i18n.tag"
          data-balloon-pos="up"
        >
          <TagIcon />
        </div>
      </button>
      <button class="switch-button" @click="setActive('timeline')">
        <div
          class="icon-wapper"
          :class="{ active: active === 'timeline' }"
          :aria-label="i18n.timeline"
          data-balloon-pos="up"
        >
          <TimeIcon />
        </div>
      </button>
    </div>

    <!-- Article -->
    <MyTransition v-if="active === 'article'">
      <div class="sticky-article-wrapper">
        <div class="title" @click="navigate('/article/')">
          <ArticleIcon />
          <span class="num">{{ articleNumber }}</span>
          {{ i18n.article }}
        </div>
        <hr />
        <ul class="sticky-article-list">
          <MyTransition
            v-for="(article, index) in $starArticles"
            :key="article.path"
            :delay="(index + 1) * 0.08"
          >
            <li
              class="sticky-article"
              @click="navigate(article.path)"
              v-text="article.title"
            />
          </MyTransition>
        </ul>
      </div>
    </MyTransition>

    <!-- Category -->
    <MyTransition v-if="active === 'category'">
      <div class="category-wrapper">
        <div
          v-if="$category.list.length !== 0"
          class="title"
          @click="navigate('/category/')"
        >
          <CategoryIcon />
          <span class="num">{{ $category.list.length }}</span>
          {{ i18n.category }}
        </div>
        <hr />
        <MyTransition :delay="0.04">
          <CategoryList />
        </MyTransition>
      </div>
    </MyTransition>

    <!-- Tags -->
    <MyTransition v-if="active === 'tag'">
      <div class="tag-wrapper">
        <div
          v-if="$tag.list.length !== 0"
          class="title"
          @click="navigate('/tag/')"
        >
          <TagIcon />
          <span class="num">{{ $tag.list.length }}</span>
          {{ i18n.tag }}
        </div>
        <hr />
        <MyTransition :delay="0.04">
          <TagList />
        </MyTransition>
      </div>
    </MyTransition>

    <!-- Timeline -->
    <MyTransition v-if="active === 'timeline'">
      <TimelineList />
    </MyTransition>
  </div>
</template>
<script src="./BlogInfoList" />
<style lang="stylus">
@require '~@mr-hope/vuepress-shared/styles/reset'

.blog-info-list
  margin 8px auto
  padding 8px 16px

  .page &
    background var(--bgcolor)
    border-radius 6px
    box-shadow 0 1px 3px 0 var(--card-shadow-color)

    &:hover
      box-shadow 0 2px 6px 0 var(--card-shadow-color)

  .switch-wrapper
    display flex
    justify-content center
    margin-bottom 8px

    .switch-button
      button()
      width 44px
      height 44px
      margin 0 8px
      padding 4px
      color var(--grey3)

      &:focus
        outline none

      .icon-wapper
        width 20px
        height 20px
        padding 8px
        border-radius 50%
        background rgba(127, 127, 127, 0.15)

        .theme-dark &
          background rgba(255, 255, 255, 0.15)

        &:hover
          cursor pointer

        &.active
          .theme-light &
            background var(--accent-color-l10)

          .theme-dark &
            background var(--accent-color-d10)

        .icon
          width 100%
          height 100%

  .sticky-article-wrapper, .category-wrapper, .tag-wrapper
    padding 8px 0

    .title
      cursor pointer

      .icon
        position relative
        bottom -0.125rem
        width 16px
        height 16px
        margin 0 6px

      .num
        position relative
        margin 0 2px
        font-size 22px

  .sticky-article-wrapper
    .sticky-article-list
      margin 8px auto

      .sticky-article
        padding 12px 8px 4px
        border-bottom 1px dashed var(--grey14)

        &:hover
          cursor pointer
          color var(--accent-color)

  .category-wrapper
    .category-list-wrapper
      margin 8px auto

  .tag-wrapper
    .tag-list-wrapper
      margin 8px auto

  .page &
    .timeline-list-wrapper
      .content
        max-height 60vh
</style>
