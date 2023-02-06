<template>
  <Common :sidebar="false">
    <template #sidebar-bottom>
      <BlogInfo />
    </template>

    <Password v-if="isGlobalEncrypted" @password-verify="checkGlobalPassword" />
    <Password
      v-else-if="isPathEncrypted"
      @password-verify="checkPathPassword"
    />
    <main v-else class="page blog">
      <div class="blog-page-wrapper">
        <BlogPage />

        <MyTransition :delay="0.16">
          <BlogInfo />
        </MyTransition>
      </div>
    </main>
  </Common>
</template>

<script src="./Blog" />

<style lang="stylus">
.page.blog
  box-sizing border-box
  min-height 100vh
  margin 0px auto
  padding-top $navbarHeight
  padding-bottom 2rem
  background var(--bgcolor-light)
  display flex
  flex-direction column
  justify-content space-between

  @media (max-width $MQMobile)
    padding $navbarMobileHeight 1.5rem 2rem

  @media (max-width $MQMobileNarrow)
    padding-left 0
    padding-right 0

.blog-page-wrapper
  box-sizing border-box
  width 100%
  margin 0 auto
  display flex
  justify-content center
  align-items flex-start

  @media (min-width $MQMobile)
    padding 0 1rem

  @media (min-width $MQNarrow)
    padding 0 2rem

  @media (min-width $MQWide)
    padding 0
</style>
