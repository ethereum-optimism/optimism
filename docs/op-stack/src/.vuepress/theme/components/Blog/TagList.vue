<template>
  <ul class="tag-list-wrapper">
    <li
      v-for="(tag, index) in tagList"
      :key="tag.path"
      class="tag"
      :class="{ active: isActive(tag.name), [`tag${index % 9}`]: true }"
      @click="clickTag(tag.path)"
    >
      <div class="tag-name">{{ tag.name }}</div>
    </li>
  </ul>
</template>

<script src="./TagList" />

<style lang="stylus">
.tag-list-wrapper
  position relative
  z-index 2
  padding-left 0
  font-family Arial, Helvetica, sans-serif
  list-style none
  display flex
  flex-wrap wrap
  justify-content space-evenly

  .tag
    display inline-block
    position relative
    vertical-align middle
    min-width 24px
    margin 4px 6px
    padding 3px 8px
    border-radius 8px
    box-shadow 0 1px 6px 0 var(--box-shadow-color)
    color var(--white)
    font-size 12px
    text-align center
    overflow hidden
    cursor pointer
    transition background-color 0.3s, transform 0.3s

    &:hover
      cursor pointer

    &.active
      transform scale(1.1, 1.1)

@require '~@mr-hope/vuepress-shared/styles/colors.styl'

for $color, $index in $colors
  .tag-list-wrapper .tag{$index}
    .theme-light &, &
      background lighten($color, 10%)

      &:hover, &.active
        background darken($color, 5%)

    .theme-dark &
      background darken($color, 5%)

      &:hover, &.active
        background lighten($color, 10%)
</style>
