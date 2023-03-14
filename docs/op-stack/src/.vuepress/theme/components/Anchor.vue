<script src="./Anchor" />

<style lang="stylus">
$headings = 2 3 4 5 6

.anchor-place-holder
  position sticky
  top: ($navbarHeight + 2rem)
  max-width $contentWidth
  margin 0 auto
  padding 0 2.5rem
  z-index 99

  @media (max-width $MQNarrow)
    padding 0 1.5rem

  & + .theme-default-content:not(.custom)
    padding-top 0

#anchor
  display none
  position absolute
  left calc(100% + 0.5rem)
  min-width 10rem
  max-width 15rem
  max-height 85vh
  overflow-y scroll

  @media (min-width $MQWide)
    .has-anchor &
      display block

  &::-webkit-scrollbar-track-piece
    background transparent

  &::-webkit-scrollbar
    width 3px

  &::-webkit-scrollbar-thumb:vertical
    background #ddd

    .theme-dark &
      background #333

  .anchor-wrapper
    position relative
    padding-left 8px

    &::before
      content ' '
      position absolute
      top 0
      left 0px
      bottom 0
      width 2px
      background var(--border-color)
      z-index -1

    > .anchor-list
      margin 0

  .anchor-list
    padding-left 0

  .anchor
    position relative
    box-sizing border-box
    padding 1px 8px
    list-style none
    line-height 1.5

    &::before
      content ' '
      position absolute
      z-index 2
      top 0
      bottom 0
      left -8px
      width 2px
      background transparent

    &:hover
      .anchor-link
        color var(--accent-color)

    &.active
      .anchor-link
        color var(--accent-color)

      &::before
        background var(--accent-color)

    .anchor-link
      display inline-block
      vertical-align middle
      position relative
      max-width 100%
      color var(--light-grey)

      > div
        text-overflow ellipsis
        white-space nowrap
        overflow hidden

      for $heading in $headings
        &.heading{$heading}
          padding-left ($heading * 8 - 16) px
          font-size: (16 - $heading)px
</style>
