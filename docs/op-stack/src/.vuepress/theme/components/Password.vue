<template>
  <div class="password" :class="{ expand: page || isMainPage }">
    <div class="hint" :class="{ hasTried }">
      {{ hasTried ? encrypt.errorHint : encrypt.title }}
    </div>
    <div class="input">
      <input v-model="password" type="password" @keypress.enter="verify" />
      <button @click="verify">OK</button>
    </div>
  </div>
</template>

<script src="./Password" />

<style lang="stylus">
.password
  background var(--bg-color)
  height 90vh - $navbarHeight
  margin-top $navbarHeight
  text-align center
  padding-left $sidebarWidth
  display flex
  flex-direction column
  justify-content center
  align-items center

  @media (max-width $MQNarrow)
    height 90vh - $navbarMobileHeight
    margin-top $navbarMobileHeight
    padding-left $mobileSidebarWidth

  @media (max-width $MQMobile)
    padding-left 0

  &.expand
    padding-left 0 !important
    margin-top 0
    height 400px

  .hint
    margin-bottom 20px
    font-family Arial, Helvetica, sans-serif
    font-weight 600
    font-size 22px
    line-height 2

    &.hasTried
      color red
      animation-name shake
      animation-duration 0.5s
      animation-timing-function ease-out
      animation-fill-mode both

  .input
    width 80%
    max-width 600px
    display flex
    justify-content center

    input
      flex 1
      width calc(100% - 60px)
      padding-left 20px
      color var(--black) !important
      background var(--bgcolor) !important
      border 2px solid var(--accent-color)
      border-radius 22px 0 0 22px
      font-size 20px
      letter-spacing 0.5em
      line-height 2
      outline none

    button
      width 70px
      padding-right 10px
      background var(--accent-color)
      color var(--bgcolor)
      border-width 0
      border-radius 0 22px 22px 0
      font-size 20px
      line-height 2
      outline none

      &:hover
        background lighten($accentColor, 15%)

@keyframes shake
  0%, 100%
    transform translateX(0)

  10%
    transform translateX(-9px)

  20%
    transform translateX(8px)

  30%
    transform translateX(-7px)

  40%
    transform translateX(6px)

  50%
    transform translateX(-5px)

  60%
    transform translateX(4px)

  70%
    transform translateX(-3px)

  80%
    transform translateX(2px)

  90%
    transform translateX(-1px)
</style>
