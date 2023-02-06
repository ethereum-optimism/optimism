<template>
  <div class="timeline-list-wrapper">
    <div class="title" @click="navigate('/timeline/')">
      <TimeIcon />
      <span class="num">{{ $timelineItems.length }}</span>
      {{ hint }}
    </div>
    <hr />
    <div class="content">
      <ul class="timeline-list">
        <MyTransition
          v-for="(item, index) in $timeline"
          :key="index"
          :delay="0.08 * (index + 1)"
        >
          <li>
            <h3 class="year">{{ item.year }}</h3>
            <ul class="year-wrapper">
              <li
                v-for="(article, articleIndex) in item.articles"
                :key="articleIndex"
              >
                <span class="date">{{ article.frontmatter.parsedDate }}</span>
                <span class="timeline-title" @click="navigate(article.path)">
                  {{ article.title }}
                </span>
              </li>
            </ul>
          </li>
        </MyTransition>
      </ul>
    </div>
  </div>
</template>

<script src="./TimelineList" />

<style lang="stylus">
.timeline-list-wrapper
  padding 8px 0
  --dot-color #fff
  --dot-bar-color #eaecef
  --dot-border-color #ddd

  .theme-dark &
    --dot-color #444
    --dot-bar-color #333
    --dot-border-color #555

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

  .content
    overflow-y scroll
    max-height 80vh

    &::-webkit-scrollbar-track-piece
      background transparent

    .timeline-list
      position relative
      margin 0 8px
      box-sizing border-box
      list-style none

      &::after
        content ' '
        position absolute
        top 14px
        left 0
        z-index -1
        margin-left -2px
        width 4px
        height calc(100% - 14px)
        background var(--dot-bar-color)

      .year
        position relative
        margin 20px 0 0px
        color var(--text-color)
        font-size 20px
        font-weight 700

        &:before
          content ' '
          position absolute
          z-index 2
          left -20px
          top 50%
          margin-left -4px
          margin-top -4px
          width 8px
          height 8px
          background var(--dot-color)
          border 1px solid var(--dot-border-color)
          border-radius 50%

      .year-wrapper
        padding-left 0 !important

        li
          position relative
          display flex
          padding 12px 0 4px
          list-style none
          border-bottom 1px dashed var(--border-color)

          &:hover
            .date
              color var(--accent-color)

              &::before
                background var(--accent-color)
                border-color var(--dot-color)

            .title
              color var(--accent-color)

          .date
            width 36px
            line-height 32px
            display inline-block
            vertical-align bottom
            font-size 12px

            &::before
              content ' '
              position absolute
              left -19px
              top 24px
              width 6px
              height 6px
              margin-left -4px
              background var(--dot-color)
              border-radius 50%
              border 1px solid var(--dot-border-color)
              z-index 2

          .timeline-title
            line-height 32px
            font-size 14px
            cursor pointer
</style>
