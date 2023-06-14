<template>
  <div class="presentation">
    <ThemeColor v-show="false" />
    <Content :key="$route.path" class="presentation-content" />
    <div
      v-click-outside="clickOutside"
      class="menu"
      :class="{ active: showMenu }"
    >
      <button class="menu-button" @click="toggle">
        <span class="icon" />
      </button>
      <button class="back-button" @click="back">
        <svg viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
          <path
            d="M1014.749 449.156v125.688H260.626l345.64 345.64-89.239 89.237L19.307 512l497.72-497.721 89.238 89.238-345.64 345.64h754.124z"
          />
        </svg>
      </button>
      <button class="home-button" @click="home">
        <svg viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg">
          <path
            d="M780.106 420.978L506.994 147.866 233.882 420.978h.045v455.11H780.06v-455.11h.046zm90.977 90.976V876.09a91.022 91.022 0 01-91.023 91.022H233.927a91.022 91.022 0 01-91.022-91.022V511.954l-67.22 67.175-64.307-64.307 431.309-431.31c35.498-35.498 93.115-35.498 128.614 0l431.309 431.31-64.307 64.307L871.083 512z"
          />
        </svg>
      </button>
    </div>
  </div>
</template>

<script src="./Slide" />

<style lang="stylus">
@require '~@mr-hope/vuepress-shared/styles/reset'

.presentation
  .md-presentation
    min-width 100vw
    min-height 100vh

  .menu-button, .back-button, .home-button
    button()
    box-sizing content-box
    position fixed
    bottom 2rem
    width 1rem
    height 1rem
    padding 0.5rem
    border-radius 50%
    background #bbb
    color var(--white)
    outline none

    &:hover
      background var(--accent-color)

    .theme-dark &
      background #666

      &:hover
        background var(--accent-color)

  .menu-button
    left 2rem
    transition transform 0.2s ease-in-out
    vertical-align middle
    z-index 50

    &::before
      content ' '
      margin-top 0.125em

    &::after
      content ' '
      margin-bottom 0.125em

    .icon
      margin 0.2em 0

    &::before, &::after, .icon
      display block
      width 100%
      height 0.2em
      transition transform 0.2s ease-in-out
      border-radius 0.05em
      background var(--white)

  .active .menu-button
    &::before
      transform translateY(0.4em) rotate(135deg)

    .icon
      transform scale(0)

    &::after
      transform translateY(-0.4em) rotate(-135deg)

  .back-button
    left 2rem
    opacity 0
    transition left 0.2s ease-out, opacity 0.2s ease-out
    z-index 49
    fill var(--white)

  .active .back-button
    left 4.5rem
    opacity 1

  .home-button
    left 2rem
    opacity 0
    transition left 0.2s ease-out, opacity 0.2s ease-out
    z-index 48
    fill var(--white)

  .active .home-button
    left 7rem
    opacity 1
</style>
