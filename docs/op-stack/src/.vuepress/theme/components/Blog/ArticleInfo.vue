<template>
  <div v-if="author || time" class="article-info">
    <!-- Author -->
    <span v-if="author" :aria-label="authorText" data-balloon-pos="down">
      <AuthorIcon />
      <span property="author" v-text="author" />
    </span>

    <!-- Writing Date -->
    <span
      v-if="time"
      class="time"
      :aria-label="timeText"
      data-balloon-pos="down"
    >
      <CalendarIcon />
      <span property="datePublished" v-text="time" />
    </span>

    <CategoryInfo
      v-if="article.frontmatter.category"
      :category="article.frontmatter.category"
    />

    <TagInfo v-if="tags.length !== 0" :tags="tags" />

    <!-- Reading time -->
    <span
      v-if="readingTime"
      class="read-time-info"
      :aria-label="readingTimeText"
      data-balloon-pos="down"
    >
      <TimerIcon />
      <span v-text="readingTime" />
      <meta property="timeRequired" :content="readingTimeContent" />
    </span>
  </div>
</template>

<script src="./ArticleInfo" />

<style lang="stylus">
$articleInfoTextSize ?= 14px

.article-info
  color var(--dark-grey)
  font-size $articleInfoTextSize
  font-family Arial, Helvetica, sans-serif

  & > span
    display inline-block
    margin-right 0.5em
    line-height 1.8

    @media (max-width $MQMobileNarrow)
      margin-right 0.3em
      font-size 0.86rem

    &::after
      --balloon-font-size 8px
      padding 0.3em 0.6em !important

    svg
      position relative
      bottom -0.125em

    .tags-wrapper
      display inline-block

  .icon
    width 1em
    height 1em
</style>
