<template>
  <ul class="article-type-wrapper">
    <li
      v-for="type in types"
      :key="type.text"
      class="article-type"
      :class="{ active: type.path === $route.path }"
      role="navigation"
      @click="navigate(type.path)"
    >
      <span>{{ type.text }}</span>
    </li>
  </ul>
</template>

<script src="./ArticleType" />

<style lang="stylus">
.article-type-wrapper
  position relative
  padding-left 0
  font-size 18px
  font-family Arial, Helvetica, sans-serif
  font-weight 600
  display flex
  justify-content center
  align-items center
  list-style none
  z-index 2

  @media (max-width $MQMobileNarrow)
    font-size 16px

.article-type
  position relative
  vertical-align middle
  margin 0.3em 0.8em
  line-height 1.2
  cursor pointer

  &::after
    position absolute
    content ' '
    left 50%
    right 50%
    bottom -6px
    height 2px
    background var(--accent-color)
    border-radius 1px
    visibility hidden
    transition left 0.2s ease-in-out, right 0.2s ease-in-out

  span
    transition all 0.3s ease-in-out

  &.active
    position relative

    span
      display inline-block
      color var(--accent-color)
      transform scale(1.1, 1.1)

  &:hover, &.active
    &::after
      left calc(50% - 8px)
      right calc(50% - 8px)
      visibility visible
</style>
