<template>
  <nav class="nav-links">
    <!-- user links -->
    <div v-for="item in navLinks" :key="item.link" class="nav-item">
      <DropdownLink v-if="item.type === 'links'" :item="item" />
      <NavLink v-else :item="item" />
    </div>
  </nav>
</template>

<script src="./NavLinks" />

<style lang="stylus">
.nav-links
  display inline-block

  .nav-item
    position relative
    display inline-block
    line-height 2rem
    margin-left 1.5rem

    &:first-child
      margin-left 0

    > .nav-link
      color var(--dark-grey)

      &::after
        position absolute
        content ' '
        left 50%
        right 50%
        bottom 0px
        height 2px
        background var(--accent-color-l10)
        border-radius 1px
        visibility hidden
        transition left 0.2s ease-in-out, right 0.2s ease-in-out

      &.active
        color var(--accent-color)

      &:hover, &.active
        &::after
          left 0
          right 0
          visibility visible
</style>
