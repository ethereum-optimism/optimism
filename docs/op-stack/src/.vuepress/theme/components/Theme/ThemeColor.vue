<template>
  <button
    v-click-outside="clickOutside"
    class="color-button"
    :class="{ select: showMenu }"
    tabindex="-1"
    aria-hidden="true"
    @click="showMenu = !showMenu"
  >
    <svg
      class="skin-icon"
      viewBox="0 0 1024 1024"
      xmlns="http://www.w3.org/2000/svg"
    >
      <path
        d="M224 800c0 9.6 3.2 44.8 6.4 54.4 6.4 48-48 76.8-48 76.8s80 41.6 147.2 0 134.4-134.4
          38.4-195.2c-22.4-12.8-41.6-19.2-57.6-19.2C259.2 716.8 227.2 761.6 224 800zM560 675.2l-32
          51.2c-51.2 51.2-83.2 32-83.2 32 25.6 67.2 0 112-12.8 128 25.6 6.4 51.2 9.6 80 9.6 54.4 0
          102.4-9.6 150.4-32l0 0c3.2 0 3.2-3.2 3.2-3.2 22.4-16 12.8-35.2
          6.4-44.8-9.6-12.8-12.8-25.6-12.8-41.6 0-54.4 60.8-99.2 137.6-99.2 6.4 0 12.8 0 22.4
          0 12.8 0 38.4 9.6 48-25.6 0-3.2 0-3.2 3.2-6.4 0-3.2 3.2-6.4 3.2-6.4 6.4-16 6.4-16 6.4-19.2
          9.6-35.2 16-73.6 16-115.2 0-105.6-41.6-198.4-108.8-268.8C704 396.8 560 675.2 560 675.2zM224
          419.2c0-28.8 22.4-51.2 51.2-51.2 28.8 0 51.2 22.4 51.2 51.2 0 28.8-22.4 51.2-51.2 51.2C246.4
          470.4 224 448 224 419.2zM320 284.8c0-22.4 19.2-41.6 41.6-41.6 22.4 0 41.6 19.2 41.6 41.6 0
          22.4-19.2 41.6-41.6 41.6C339.2 326.4 320 307.2 320 284.8zM457.6 208c0-12.8 12.8-25.6 25.6-25.6
          12.8 0 25.6 12.8 25.6 25.6 0 12.8-12.8 25.6-25.6 25.6C470.4 233.6 457.6 220.8 457.6 208zM128
          505.6C128 592 153.6 672 201.6 736c28.8-60.8 112-60.8 124.8-60.8-16-51.2 16-99.2
          16-99.2l316.8-422.4c-48-19.2-99.2-32-150.4-32C297.6 118.4 128 291.2 128 505.6zM764.8
          86.4c-22.4 19.2-390.4 518.4-390.4 518.4-22.4 28.8-12.8 76.8 22.4 99.2l9.6 6.4c35.2 22.4
          80 12.8 99.2-25.6 0 0 6.4-12.8 9.6-19.2 54.4-105.6 275.2-524.8 288-553.6
          6.4-19.2-3.2-32-19.2-32C777.6 76.8 771.2 80 764.8 86.4z"
      />
    </svg>
    <transition mode="out-in" name="menu-transition">
      <div v-show="showMenu" class="color-picker-menu">
        <ThemeOptions />
      </div>
    </transition>
  </button>
</template>

<script src="./ThemeColor" />

<style lang="stylus">
@require '~@mr-hope/vuepress-shared/styles/reset'

.color-button
  button()
  position relative
  width 2.25rem
  height 2.25rem
  margin 0 0.25rem
  padding 0.5rem
  outline none
  color #aaa
  flex-shrink 0

  &:hover, &.select
    color var(--accent-color)

  &.select:hover
    color #aaa

  .skin-icon
    width 100%
    height 100%
    fill currentcolor

  .color-picker-menu
    position absolute
    top: $navbarHeight - $navbarVerticalPadding
    left 50%
    min-width 100px
    margin 0
    padding 0.5em 0.75em
    background var(--bgcolor)
    box-shadow 2px 2px 10px var(--card-shadow-color)
    color var(--dark-grey)
    border-radius 0.25em
    transform translateX(-50%)
    z-index 250

    @media (max-width $MQMobile)
      top: $navbarMobileHeight - $navbarMobileVerticalPadding
      transform translateX(-80%)

    &::before
      content ''
      position absolute
      top -7px
      left 50%
      border-style solid
      border-color transparent transparent var(--bgcolor)
      border-width 0 7px 7px
      transform translateX(-50%)

      @media (max-width $MQMobile)
        left 80%

    &.menu-transition-enter-active, &.menu-transition-leave-active
      transition all 0.25s ease-in-out

    &.menu-transition-enter, &.menu-transition-leave-to
      top 30px
      opacity 0

    ul
      list-style-type none
      margin 0
      padding 0

@media (max-width $MQMobile)
  .color-picker
    .color-picker-menu
      left calc(50% - 35px)

      &::before
        left calc(50% + 35px)
</style>
