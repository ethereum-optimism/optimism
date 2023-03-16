<template>
  <nav class="sidebar-nav-links">
    <!-- user links -->
    <div v-for="item in navLinks" :key="item.link" class="nav-item">
      <SidebarDropdownLink v-if="item.type === 'links'" :item="item" />
      <NavLink v-else :item="item" />
    </div>

    <RepoLink />
  </nav>
</template>

<script src="./SidebarNavLinks" />

<style lang="stylus">
.sidebar-nav-links
  display none
  padding 0.5rem 0 0.75rem 0
  border-bottom 1px solid var(--border-color)

  @media (max-width $MQMedium)
    display block

  ul
    padding 0
    margin 0.25rem 0 0
    list-style-type none

  .nav-item
    position relative
    display block
    padding 0.5rem 0 0.5rem 1.5rem
    font-size 1.1em
    line-height 1.25rem

    &:first-child
      margin-left 0
</style>
