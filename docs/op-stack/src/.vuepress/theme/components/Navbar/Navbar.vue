<template>
  <header class="navbar" :class="{ 'can-hide': canHide }">
    <slot name="start" />

    <SidebarButton @toggle-sidebar="$emit('toggle-sidebar')" />

    <RouterLink ref="siteInfo" :to="$localePath" class="home-link">
      <img
        v-if="siteBrandLogo"
        class="logo"
        :class="{ light: Boolean(siteBrandDarkLogo) }"
        :src="siteBrandLogo"
        :alt="siteBrandTitle"
      />
      <img
        v-if="siteBrandDarkLogo"
        class="logo dark"
        :src="siteBrandDarkLogo"
        :alt="siteBrandTitle"
      />
      <span
        v-if="siteBrandTitle"
        class="site-name"
        :class="{ 'can-hide': canHideSiteBrandTitle }"
        >{{ siteBrandTitle }}</span
      >
    </RouterLink>

    <slot name="center" />

    <div
      :style="
        linksWrapMaxWidth ? { 'max-width': `${linksWrapMaxWidth}px` } : {}
      "
      class="links"
    >
      <ThemeColor />
      <AlgoliaSearchBox v-if="isAlgoliaSearch" :options="algoliaConfig" />
      <SearchBox
        v-else-if="
          $themeConfig.search !== false && $page.frontmatter.search !== false
        "
      />
      <NavLinks class="can-hide" />
      <LanguageDropdown />
      <RepoLink class="can-hide" />

      <slot name="end" />
    </div>
  </header>
</template>

<script src="./Navbar" />

<style lang="stylus">
.navbar
  position fixed
  z-index 200
  top 0
  left 0
  right 0
  height $navbarHeight
  padding $navbarVerticalPadding $navbarHorizontalPadding
  background var(--bgcolor-blur)
  box-sizing border-box
  box-shadow 0 2px 8px var(--card-shadow-color)
  backdrop-filter saturate(200%) blur(20px)
  line-height: $navbarHeight - $navbarVerticalPadding * 2
  transition transform 0.3s ease-in-out

  @media (max-width $MQMedium)
    height $navbarMobileHeight
    padding $navbarMobileVerticalPadding $navbarMobileHorizontalPadding
    padding-left: $navbarMobileHorizontalPadding + 2.4rem
    line-height: $navbarMobileHeight - $navbarMobileVerticalPadding * 2

  .hide-navbar &.can-hide
    transform translateY(-100%)

  a, span, img
    display inline-block

  .logo
    min-width: 200px
    height: $navbarHeight - $navbarVerticalPadding * 2
    margin-right 0.8rem
    vertical-align top

    @media (max-width $MQMedium)
      min-width: $navbarMobileHeight - $navbarMobileVerticalPadding * 2
      height: $navbarMobileHeight - $navbarMobileVerticalPadding * 2

    .theme-light &
      &.light
        display inline-block

      &.dark
        display none

    .theme-dark &
      &.light
        display none

      &.dark
        display inline-block

  .can-hide
    @media (max-width $MQMedium)
      display none

  .site-name
    font-size 1.5rem
    color var(--text-color)
    position relative

    @media (max-width $MQMedium)
      width calc(100vw - 9.4rem)
      overflow hidden
      white-space nowrap
      text-overflow ellipsis

  .links
    position absolute
    top $navbarVerticalPadding
    right $navbarHorizontalPadding
    display flex
    box-sizing border-box
    padding-left 1.5rem
    font-size 0.9rem
    white-space nowrap

    @media (max-width $MQMedium)
      padding-left 0
      top $navbarMobileVerticalPadding
      right $navbarMobileHorizontalPadding
</style>
