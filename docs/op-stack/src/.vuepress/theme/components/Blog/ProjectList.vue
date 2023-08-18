<template>
  <div class="project-list">
    <div
      v-for="(project, index) in $frontmatter.project || []"
      :key="project.name"
      class="project"
      :class="`project${index % 9}`"
      @click="navigate(project.link)"
    >
      <div
        v-if="project.cover"
        class="cover"
        :style="`background: url(${$withBase(
          project.cover
        )}) center/cover no-repeat;`"
      />
      <component :is="`${project.type}-icon`" />
      <div class="name">{{ project.name }}</div>
      <div class="desc">{{ project.desc }}</div>
    </div>
  </div>
</template>

<script src="./ProjectList" />

<style lang="stylus">
.project-list
  position relative
  display flex
  justify-content flex-start
  align-content stretch
  align-items stretch
  flex-wrap wrap
  font-family sans-serif
  margin-bottom 12px
  z-index 2

  .project
    position relative
    width calc(50% - 40px)
    background-color var(--grey14)
    border-radius 8px
    margin 6px 8px
    padding 12px
    transition background-color 0.3s, transform 0.3s

    @media (min-width $MQNarrow)
      width calc(33% - 40px)

    @media (min-width $MQWide)
      width calc(25% - 40px)

    &:hover
      cursor pointer
      transform scale(0.98, 0.98)

    .cover
      content ''
      opacity 0.5
      top 0
      left 0
      bottom 0
      right 0
      position absolute
      z-index 1

    .icon
      position relative
      z-index 2
      float right
      width 20px
      height 20px

    .name
      position relative
      z-index 2
      color var(--grey3)
      font-size 16px
      font-weight 500

    .desc
      position relative
      z-index 2
      margin 6px 0
      color var(--dark-grey)
      font-size 13px

@require '~@mr-hope/vuepress-shared/styles/colors.styl'

for $color, $index in $colors
  .project-list .project{$index}
    &, .theme-light &
      background lighten($color, 90%)

      &:hover
        background lighten($color, 75%)

    .theme-dark &
      background darken($color, 75%)

      &:hover
        background darken($color, 60%)
</style>
