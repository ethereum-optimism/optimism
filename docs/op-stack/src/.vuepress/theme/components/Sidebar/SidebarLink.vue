<script src="./SidebarLink" />

<style lang="stylus">
$headings = 3 4 5 6

.sidebar-links
  ul
    padding 0
    margin 0
    list-style-type none

    &.sidebar-sub-headers
      padding-left 1rem
      font-size 0.95em

      .has-anchor &
        @media (min-width $MQWide)
          display none

a.sidebar-link
  display inline-block
  box-sizing border-box
  width 100%
  padding 0.35rem 1rem 0.35rem 1.25rem
  border-left 0.2rem solid transparent
  color var(--text-color)
  font-size 1em
  line-height 1.5
  font-weight 400

  for $heading in $headings
    &.heading{$heading}
      padding-left ($heading - 0.5) * 0.5rem !important

  .iconfont
    margin-right 0.25em

  &:hover
    background-color var(--bgcolor-light)

  &.active
    border-left-color var(--accent-color)
    background-color var(--accent-color-a15)
    color var(--accent-color)
    font-weight 500

    .iconfont
      color var(--accent-color)

  .sidebar-group &
    padding-left 1.75rem

  .sidebar-sub-headers &
    padding-top 0.25rem
    padding-bottom 0.25rem
    border-left none

    &.active
      background-color transparent
      font-weight 500
</style>
