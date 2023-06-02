<template>
  <div v-if="mediaLink" class="media-links-wrapper">
    <a
      v-for="link in links"
      :key="link.name"
      class="media-link"
      :href="link.url"
      rel="noopener noreferrer"
      target="_blank"
      :aria-label="link.icon"
      data-balloon-pos="up"
    >
      <span class="sr-only" v-text="link.icon" />
      <Component :is="link.icon" />
    </a>
  </div>
</template>

<script src="./MediaLinks" />

<style lang="stylus">
.media-links-wrapper
  display flex
  justify-content center
  flex-wrap wrap
  margin 8px auto

  .media-link
    width 26px
    height 26px
    margin 4px
    transform scale(1, 1)
    transition transform 0.18s ease-out 0.18s

    &:hover
      cursor pointer
      transform scale(1.2, 1.2)

    &::after
      --balloon-font-size 8px
      padding 0.3em 0.6em

    .icon
      width 100%
      height 100%
</style>
