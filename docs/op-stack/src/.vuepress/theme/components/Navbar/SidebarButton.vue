<template>
  <button
    class="sidebar-button"
    title="Sidebar Button"
    @click="$emit('toggle-sidebar')"
  >
    <span class="icon" />
  </button>
</template>

<style lang="stylus">
@require '~@mr-hope/vuepress-shared/styles/reset'

.sidebar-button
  button()
  display none
  box-sizing content-box
  position absolute
  top calc(50% - 1.075rem)
  left 1rem
  width 1.25rem
  height 1.25rem
  padding 0.45rem
  font unset
  vertical-align middle
  transition transform 0.2s ease-in-out

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
    background var(--text-color)

.sidebar-open .sidebar-button
  &::before
    transform translateY(0.4em) rotate(135deg)

  .icon
    transform scale(0)

  &::after
    transform translateY(-0.4em) rotate(-135deg)

@media (max-width $MQMedium)
  .sidebar-button
    display block
</style>
