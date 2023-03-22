<template>
  <main
    :aria-labelledby="$frontmatter.heroText !== null ? 'main-title' : null"
    class="home"
  >
    <header class="hero">
      <MyTransition>
        <img
          v-if="$frontmatter.heroImage"
          key="light"
          :class="{ light: Boolean($frontmatter.darkHeroImage) }"
          :src="$withBase($frontmatter.heroImage)"
          :alt="$frontmatter.heroAlt || 'HomeLogo'"
        />
      </MyTransition>
      <MyTransition>
        <img
          v-if="$frontmatter.darkHeroImage"
          key="dark"
          class="dark"
          :src="$withBase($frontmatter.darkHeroImage)"
          :alt="$frontmatter.heroAlt || 'HomeLogo'"
        />
      </MyTransition>
      <div class="hero-info">
        <MyTransition :delay="0.04">
          <h1
            v-if="$frontmatter.heroText !== false"
            id="main-title"
            v-text="$frontmatter.heroText || $title || 'Hello'"
          />
        </MyTransition>
        <MyTransition :delay="0.08">
          <p
            class="description"
            v-text="
              $frontmatter.tagline ||
              $description ||
              'Welcome to your VuePress site'
            "
          />
        </MyTransition>
        <MyTransition :delay="0.12">
          <p v-if="$frontmatter.action" class="action">
            <NavLink
              v-for="action in actionLinks"
              :key="action.text"
              :item="action"
              class="action-button"
              :class="action.type || ''"
            />
          </p>
        </MyTransition>
      </div>
    </header>

    <MyTransition :delay="0.16">
      <div>
        <h2 class="features-header">Resources</h2>
        <div
          v-if="$frontmatter.features && $frontmatter.features.length"
          class="features"
        >
          <template v-for="(feature, index) in $frontmatter.features">
            <div
              v-if="feature.link"
              :key="index"
              class="feature link"
              :class="`feature${index % 9}`"
              tabindex="0"
              role="navigation"
              @click="navigate(feature.link)"
            >
              <div class="icon-container">
                <i :class="`far fa-${feature.icon}`"></i>
              </div>
              <h2>{{ feature.title }}</h2>
              <p>{{ feature.details }}</p>
            </div>
            <div
              v-else
              :key="index"
              class="feature"
              :class="`feature${index % 9}`"
            >
              <h2>{{ feature.title }}</h2>
              <p>{{ feature.details }}</p>
            </div>
          </template>
        </div>
      </div>
    </MyTransition>

    <MyTransition :delay="0.24">
      <Content class="theme-default-content custom" />
    </MyTransition>
  </main>
</template>

<script src="./Home" />

<style lang="stylus">
.home
  display block
  max-width $homePageWidth
  min-height 100vh - $navbarHeight
  padding $navbarHeight 2rem 0
  margin 0px auto
  overflow-x hidden

  @media (max-width $MQNarrow)
    min-height 100vh - $navbarMobileHeight
    padding-top $navbarMobileHeight

  @media (max-width $MQMobileNarrow)
    padding-left 1.5rem
    padding-right 1.5rem

  .hero
    text-align center

    @media (min-width $MQNarrow)
      display flex
      justify-content space-evenly
      align-items center
      text-align left

    img
      display block
      max-width 100%
      max-height 320px
      margin 0

      @media (max-width $MQNarrow)
        max-height 280px
        margin 3rem auto 1.5rem

      @media (max-width $MQMobile)
        max-height 240px
        margin 2rem auto 1.2rem

      @media (max-width $MQMobileNarrow)
        max-height 210px
        margin 1.5rem auto 1rem

      .theme-light &
        &.light
          display block

        &.dark
          display none

      .theme-dark &
        &.light
          display none

        &.dark
          display block

    h1
      font-size 3rem

      @media (max-width $MQMobile)
        font-size 2.5rem

      @media (max-width $MQMobileNarrow)
        font-size 2rem

    h1, .description, .action
      margin 1.8rem auto

      @media (max-width $MQMobile)
        margin 1.5rem auto

      @media (max-width $MQMobileNarrow)
        margin 1.2rem auto

    .description
      max-width 35rem
      color var(--text-color-l40)
      font-size 1.6rem
      line-height 1.3

      @media (max-width $MQMobile)
        font-size 1.4rem

      @media (max-width $MQMobileNarrow)
        font-size 1.2rem

    .action-button
      display inline-block
      margin 0.6rem 0.8rem
      padding 1rem 1.5rem
      border 2px solid var(--accent-color)
      border-radius 2rem
      color var(--accent-color)
      font-size 1.2rem
      transition background 0.1s ease
      overflow hidden

      @media (max-width $MQMobile)
        padding 0.8rem 1.2rem
        font-size 1.1rem

      @media (max-width $MQMobileNarrow)
        font-size 1rem

      &:hover
        color var(--white)
        background-color var(--accent-color)

      &.primary
        color var(--white)
        background-color var(--accent-color)

        &:hover
          border-color var(--accent-color-l10)
          background-color var(--accent-color-l10)

        .theme-dark &
          &:hover
            border-color var(--accent-color-d10)
            background-color var(--accent-color-d10)

  .features
    display flex
    flex-wrap wrap
    justify-content center
    align-items stretch
    align-content stretch
    margin 0 -2rem
    padding 1.2rem 0
    border-top 1px solid var(--border-color)

    @media (max-width $MQMobileNarrow)
      margin 0 -1.5rem

    .feature
      display flex
      flex-direction column
      justify-content center
      flex-basis calc(33% - 4rem)
      margin 0.5rem
      padding 0 1.5rem
      border-radius 0.5rem
      transition transform 0.3s, box-shadow 0.3s
      overflow hidden

      @media (max-width $MQNarrow)
        flex-basis calc(50% - 4rem)

      @media (max-width $MQMobile)
        font-size 0.95rem

      @media (max-width $MQMobileNarrow)
        flex-basis calc(100%)
        font-size 0.9rem
        margin 0.5rem 0
        border-radius 0

      &.link
        cursor pointer

      &:hover
        box-shadow 0 2px 12px 0 var(--card-shadow-color)

      h2
        margin-bottom 0.25rem
        border-bottom none
        color var(--text-color-l10)
        font-size 1.25rem
        font-weight 500

        @media (max-width $MQMobileNarrow)
          font-size 1.2rem

      p
        margin-top 0
        color var(--text-color-l25)

  {$contentClass}
    padding-bottom 1.5rem

@require '~@mr-hope/vuepress-shared/styles/colors.styl'

for $color, $index in $colors
  .home .features .feature{$index}
    &, .theme-light &
      background lighten($color, 90%)

    .theme-dark &
      background darken($color, 75%)
</style>
