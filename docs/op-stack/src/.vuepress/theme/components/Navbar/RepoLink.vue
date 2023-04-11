<template>
  <!-- repo link -->
  <a
    v-if="repoLink && $themeConfig.repoDisplay !== false"
    class="repo-link"
    rel="noopener noreferrer"
    :href="repoLink"
    target="_blank"
  >
    {{ repoLabel }}
    <OutboundLink />
  </a>
</template>

<script src="./RepoLink" />

<style lang="stylus">
.repo-link
  .navbar &
    color var(--dark-grey)
    margin-left 1rem

  .sidebar-nav-links &
    display block
    padding 0.5rem 0 0.5rem 1.5rem
    font-size 1.1em
    line-height 1.25rem
</style>
