<template>
  <ul class="category-list-wrapper">
    <li
      v-for="(category, index) in $category.list"
      :key="category.path"
      class="category"
      :class="{
        active: category.path === $route.path,
        [`category${index % 9}`]: true,
      }"
      @click="clickCategory(category.path)"
    >
      {{ capitalize(category.name) }}
      <span class="category-num">{{ category.pages.length }}</span>
    </li>
  </ul>
</template>

<script src="./CategoryList" />

<style lang="stylus">
$categoryListTextSize ?= 14px

.category-list-wrapper
  position relative
  z-index 2
  padding-left 0
  font-size $categoryListTextSize
  font-family Arial, Helvetica, sans-serif
  list-style none

  .category
    display inline-block
    vertical-align middle
    margin 0.3rem 0.6rem 0.8rem
    padding 0.4rem 0.8rem
    border-radius 0.25rem
    box-shadow 0 1px 4px 0 var(--card-shadow-color)
    color var(--dark-grey)
    cursor pointer
    overflow hidden
    transition background-color 0.3s, color 0.3s

    @media (max-width $MQMobileNarrow)
      font-size 0.9rem

    .category-num
      display inline-block
      min-width 1rem
      height 1.2rem
      margin-left 0.2em
      padding 0 0.1rem
      border-radius 0.6rem
      color var(--white)
      font-family sans-serif
      font-size 0.7rem
      line-height 1.2rem
      text-align center

@require '~@mr-hope/vuepress-shared/styles/colors.styl'

for $color, $index in $colors
  .category-list-wrapper .category{$index}
    &, .theme-light &
      background lighten($color, 90%)

      &:hover
        background lighten($color, 75%)

      &.active
        background var(--accent-color)
        color var(--white)

        .category-num
          color var(--accent-color)
          background var(--bgcolor-light)

    .theme-dark &
      background darken($color, 75%)

      &:hover
        background darken($color, 60%)

      &.active
        background var(--accent-color-d10)
        color var(--white)

    .category-num
      background $color
</style>
