<template>
  <div class="timeline-wrapper">
    <ul class="timeline-content">
      <MyTransition>
        <li class="desc">{{ hint }}</li>
      </MyTransition>
      <Anchor :items="anchorConfig" />
      <MyTransition
        v-for="(item, index) in $timeline"
        :key="index"
        :delay="0.08 * (index + 1)"
      >
        <li>
          <h3 :id="item.year" class="year">
            <span>{{ item.year }}</span>
          </h3>
          <ul class="year-wrapper">
            <li
              v-for="(article, articleIndex) in item.articles"
              :key="articleIndex"
            >
              <span class="date">{{ article.frontmatter.parsedDate }}</span>
              <span class="title" @click="navigate(article.path)">
                {{ article.title }}
              </span>
            </li>
          </ul>
        </li>
      </MyTransition>
    </ul>
  </div>
</template>

<script src="./Timeline" />

<style lang="stylus">
.timeline-wrapper
  max-width 740px
  margin 0 auto
  padding 40px 0
  --dot-color #fff
  --dot-bar-color #eaecef
  --dot-border-color #ddd

  .theme-dark &
    --dot-color #444
    --dot-bar-color #333
    --dot-border-color #555

  #anchor
    left unset
    right 0
    min-width 0

  .anchor-wrapper
    position relative
    z-index 10

  .timeline-content
    box-sizing border-box
    position relative
    padding-left 76px
    list-style none

    &::after
      content ' '
      position absolute
      top 14px
      left 64px
      z-index -1
      width 4px
      height calc(100% - 38px)
      margin-left -2px
      background var(--dot-bar-color)

    .desc
      position relative
      color var(--text-color)
      font-size 18px

      @media (min-width $MQNormal)
        font-size 20px

      &:before
        content ' '
        position absolute
        z-index 2
        left -12px
        top 50%
        width 8px
        height 8px
        margin-left -6px
        margin-top -6px
        background var(--dot-color)
        border 2px solid var(--dot-border-color)
        border-radius 50%

    .year
      margin-top 0.5rem - $navbarHeight
      margin-bottom 0.5rem
      padding-top: ($navbarHeight + 3rem)
      color var(--text-color)
      font-size 26px
      font-weight 700

      span
        position relative

        &:before
          content ' '
          position absolute
          z-index 2
          left -12px
          top 50%
          width 8px
          height 8px
          margin-left -6px
          margin-top -6px
          background var(--dot-color)
          border 2px solid var(--dot-border-color)
          border-radius 50%

    .year-wrapper
      padding-left 0 !important

      li
        position relative
        display flex
        padding 30px 0 10px
        border-bottom 1px dashed var(--border-color)
        list-style none

        &:hover
          cursor pointer

          .date
            font-size 16px
            transition font-size 0.3s ease-out

            &::before
              background-color var(--bgcolor)
              border-color var(--accent-color)

          .title
            color var(--accent-color)
            font-size 18px
            transition font-size 0.3s ease-out

        .date
          position absolute
          right calc(100% + 24px)
          text-align right
          width 40px
          font-size 14px
          line-height 30px

          &::before
            content ' '
            position absolute
            z-index 2
            right -16px
            top 50%
            width 6px
            height 6px
            margin-left -6px
            margin-top -6px
            background var(--dot-color)
            border 2px solid var(--dot-border-color)
            border-radius 50%

        .title
          position relative
          font-size 16px
          line-height 30px

@media (max-width $MQMobile)
  .timeline-wrapper
    margin 0 1.2rem
</style>
D
