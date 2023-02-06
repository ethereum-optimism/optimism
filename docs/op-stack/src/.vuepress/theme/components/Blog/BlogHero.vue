<template>
  <div
    v-if="$frontmatter.hero !== false"
    class="blog-hero"
    :class="{ full: $frontmatter.heroFullScreen }"
    :style="{ ...bgImageStyle }"
  >
    <div
      class="mask"
      :style="{
        background: `url(${
          $frontmatter.bgImage
            ? $withBase($frontmatter.bgImage)
            : defaultHeroImage
        }) center/cover no-repeat`,
      }"
    />
    <MyTransition :delay="0.04">
      <img
        v-if="$frontmatter.heroImage"
        class="hero-logo"
        :style="heroImageStyle || {}"
        :src="$withBase($frontmatter.heroImage)"
        alt="hero"
      />
    </MyTransition>
    <MyTransition :delay="0.08">
      <h1 v-if="$frontmatter.showTitle !== false">
        {{ $frontmatter.heroText || $title || "Hope" }}
      </h1>
    </MyTransition>

    <MyTransition :delay="0.12">
      <p v-if="$description" class="description" v-text="$description" />
    </MyTransition>
  </div>
</template>

<script src="./BlogHero" />

<style lang="stylus">
.blog-hero
  position relative
  color #eee
  margin-bottom 16px
  height 450px
  display flex
  flex-direction column
  justify-content center

  @media (max-width $MQMobile)
    height 350px
    margin 0 -1.5rem 16px

  @media (max-width $MQMobileNarrow)
    margin 0 0 16px

  &.full
    height 'calc(100vh - %s)' % $navbarHeight !important

    @media (max-width $MQMobile)
      height 'calc(100vh - %s)' % $navbarMobileHeight !important

    .mask
      background-position-y top !important

  .mask
    position absolute
    top 0
    bottom 0
    left 0
    right 0

    &:after
      display block
      content ' '
      background var(--light-grey)
      position absolute
      top 0
      bottom 0
      left 0
      right 0
      z-index 1
      opacity 0.2

  & > :not(.mask)
    position relative
    z-index 2

  h1
    margin 0.5rem auto
    font-size 36px

    @media (max-width $MQNarrow)
      font-size 30px

    @media (max-width $MQMobile)
      font-size 36px

    @media (max-width $MQMobileNarrow)
      font-size 30px

  .hero-logo + h1
    margin 0 auto

  .description
    margin 1.2rem auto 0
    font-size 20px

    @media (max-width $MQNarrow)
      font-size 18px

    @media (max-width $MQMobile)
      font-size 20px

    @media (max-width $MQMobileNarrow)
      font-size 18px
</style>
