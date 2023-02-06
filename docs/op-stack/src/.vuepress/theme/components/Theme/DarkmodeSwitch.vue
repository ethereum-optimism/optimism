<template>
  <div class="darkmode-switch">
    <template v-if="darkmodeConfig === 'auto-switch'">
      <div
        class="item day"
        :class="{ active: darkmode === 'off' }"
        @click="setDarkmode('off')"
      >
        <LightIcon />
      </div>
      <div
        class="item auto"
        :class="{ active: darkmode === 'auto' }"
        @click="setDarkmode('auto')"
      >
        <AutoIcon />
      </div>
      <div
        class="item night"
        :class="{ active: darkmode === 'on' }"
        @click="setDarkmode('on')"
      >
        <DarkIcon />
      </div>
    </template>
    <div v-else-if="darkmodeConfig === 'switch'" class="switch">
      <input
        id="switch"
        class="switch-input"
        type="checkbox"
        :checked="darkmode !== 'on'"
        @click="setDarkmode(darkmode === 'on' ? 'off' : 'on')"
      />
      <label class="label" for="switch">
        <span class="label-content" />
      </label>
    </div>
  </div>
</template>

<script src="./DarkmodeSwitch" />

<style lang="stylus">
@keyframes starry_star
  50%
    background rgba(255, 255, 255, 0.1)
    box-shadow #fff 7.5px -0.75px 0 0, #fff 3px 2.5px 0 -0.25px, rgba(255, 255, 255, 0.1) 9.5px 4.5px 0 0.25px, #fff 8px 8.5px 0 0, rgba(255, 255, 255, 0.1) 5px 6px 0 -0.375px, #fff 1.25px 9.5px 0 0.25px

@keyframes bounceIn
  0%
    opacity 0
    transform scale(0.3)

  50%
    opacity 100
    transform scale(1.1)

  55%
    transform scale(1.1)

  75%
    transform scale(0.9)

  100%
    opacity 100
    transform scale(1)

.darkmode-switch
  display flex
  height 22px

  &:hover
    cursor pointer

  .item
    padding 2px
    border 1px solid var(--accent-color)
    border-left none
    line-height 1

    &:first-child
      border-left 1px solid var(--accent-color)

    &.day
      border-radius 4px 0 0 4px

    &.night
      border-radius 0 4px 4px 0

    .icon
      width 16px
      height 16px
      color var(--accent-color)

    &.active
      background var(--accent-color)

      &:hover
        cursor default

      .icon
        color var(--white)

  .switch
    display block
    text-align center
    user-select none

    .label
      display block
      position relative
      width 31.25px
      height 17.5px
      margin 0 auto
      border-radius 17.5px
      border 1px solid #1c1c1c
      background #3c4145
      font-size 1.4em
      transition all 250ms ease-in

      &:hover
        cursor pointer

      &:before
        content ''
        display block
        position absolute
        top 0.5px
        left 1px
        width 14px
        height 14px
        border 1.25px solid #e3e3c7
        border-radius 50%
        background #fff
        transition all 250ms ease-in

      &:after
        content ''
        display block
        position absolute
        top 62%
        left 9.75px
        z-index 10
        width 2.8px
        height 2.8px
        opacity 0
        background #fff
        border-radius 50%
        box-shadow #fff 0 0, #fff 0.75px 0, #fff 1.5px 0, #fff 2.25px 0, #fff 2.75px 0, #fff 3.5px 0, #fff 4px 0, #fff 5.25px -0.25px 0 0.25px, #fff 4px -1.75px 0 -0.5px, #fff 1.75px -1.75px 0 0.25px, #d3d3d3 0 0 0 1px, #d3d3d3 1.5px 0 0 1px, #d3d3d3 2.75px 0 0 1px, #d3d3d3 4px 0 0 1px, #d3d3d3 5.25px -0.25px 0 1.25px, #d3d3d3 4px -1.75px 0 0.25px, #d3d3d3 1.75px -1.75px 0 1.25px
        transition opacity 100ms ease-in

      .label-content
        display block
        position absolute
        top 2.25px
        left 52.5%
        z-index 20
        width 1px
        height 1px
        border-radius 50%
        background #fff
        box-shadow rgba(255, 255, 255, 0.1) 7.5px -0.75px 0 0, rgba(255, 255, 255, 0.1) 3px 2.5px 0 -0.25px, #fff 9.5px 4.5px 0 0.25px, rgba(255, 255, 255, 0.1) 8px 8.5px 0 0, #fff 5px 6px 0 0.375px, rgba(255, 255, 255, 0.1) 1.25px 9.5px 0 0.25px
        animation starry_star 5s ease-in-out infinite
        transition all 250ms ease-in

        &:before
          content ''
          display block
          position absolute
          top -0.5px
          left -6.25px
          width 4.5px
          height 4.5px
          background #fff
          border-radius 50%
          border 1.25px solid #e3e3c7
          box-shadow #e3e3c7 -7px 0 0 -0.75px, #e3e3c7 -2px 6px 0 -0.5px
          transform-origin -1.5px 130%
          transition all 250ms ease-in

    .switch-input
      display none
      transition all 250ms ease-in

      &:checked + .label
        background #9ee3fb
        border 1px solid #86c3d7

        &:before
          left 13.75px
          background #ffdf6d
          border 1.25px solid #e1c348

        &:after
          opacity 100
          animation bounceIn 0.6s ease-in-out 0.1s
          animation-fill-mode backwards

        & > .label-content
          opacity 0
          box-shadow rgba(255, 255, 255, 0.1) 7.5px -0.75px 0 -1px, rgba(255, 255, 255, 0.1) 3px 2.5px 0 -1.25px, #fff 9.5px 4.5px 0 -0.75px, rgba(255, 255, 255, 0.1) 8px 8.5px 0 -1px, #fff 5px 6px 0 -1.375px, rgba(255, 255, 255, 0.1) 1.25px 9.5px 0 -0.75px
          animation none

          &:before
            left 6.25px
            transform rotate(70deg)
</style>
