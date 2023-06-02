<template>
  <div class="page blog">
    <BlogHero />

    <div class="blog-page-wrapper">
      <main class="blog-home">
        <MyTransition :delay="0.16">
          <ProjectList />
        </MyTransition>

        <MyTransition :delay="0.24">
          <ArticleList :key="$route.path" />
        </MyTransition>
      </main>

      <MyTransition :delay="0.16">
        <BlogInfo />
      </MyTransition>
    </div>

    <MyTransition :delay="0.28">
      <Content :key="$route.path" class="theme-default-content" custom />
    </MyTransition>
  </div>
</template>

<script src="./BlogHome" />

<style lang="stylus">
.page.blog
  box-sizing content-box
  min-height 100vh
  padding-top $navbarHeight
  padding-bottom 2rem
  margin 0px auto
  background var(--bgcolor-light)

  @media (max-width $MQMobile)
    padding $navbarMobileHeight 1.5rem 2rem

  @media (max-width $MQMobileNarrow)
    padding-left 0
    padding-right 0

  .blog-page-wrapper
    display flex
    justify-content center
    align-items flex-start
    margin 0 auto

    @media (min-width $MQMobile)
      padding 0 1rem

    @media (min-width $MQNarrow)
      padding 0 2rem

    @media (min-width $MQWide)
      padding 0

    .blog-home
      max-width 780px
      overflow hidden
      flex 1

      @media (min-width $MQMobile)
        margin 0 15px

  .theme-default-content:empty
    padding 0
</style>
