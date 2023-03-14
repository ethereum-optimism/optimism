<template>
  <div id="docsearch" />
</template>

<script src="./Full" />

<style lang="stylus">
@keyframes fade-in
  0%
    opacity 0

  to
    opacity 1

body
  --docsearch-spacing 12px
  --docsearch-icon-stroke-width 1.4
  --docsearch-muted-color #969faf
  --docsearch-container-background rgba(101, 108, 133, 0.8)
  --docsearch-modal-width 560px
  --docsearch-modal-height 600px
  --docsearch-modal-shadow inset 1px 1px 0 0 hsla(0, 0%, 100%, 0.5), 0 3px 8px 0 #555a64
  --docsearch-searchbox-height 56px
  --docsearch-searchbox-background #ebedf0
  --docsearch-searchbox-focus-background #efeef4
  --docsearch-searchbox-shadow inset 0 0 0 2px var(--accent-color)
  --docsearch-hit-height 56px
  --docsearch-hit-color var(--dark-grey)
  --docsearch-hit-shadow 0 1px 3px 0 #d4d9e1
  --docsearch-key-gradient linear-gradient(-225deg, #d5dbe4, #f8f8f8)
  --docsearch-key-shadow inset 0 -2px 0 0 #cdcde6, inset 0 0 1px 1px #fff, 0 1px 2px 1px rgba(30, 35, 90, 0.4)
  --docsearch-footer-height 44px
  --docsearch-footer-shadow 0 -1px 0 0 #e0e3e8, 0 -3px 6px 0 rgba(69, 98, 155, 0.12)

  @media (max-width $MQMobile)
    --docsearch-searchbox-height 44px
    --docsearch-spacing 8px
    --docsearch-footer-height 40px

body.theme-dark
  --docsearch-container-background rgba(9, 10, 17, 0.8)
  --docsearch-modal-shadow inset 1px 1px 0 0 #2c2e40, 0 3px 8px 0 #000309
  --docsearch-searchbox-background #090a11
  --docsearch-searchbox-focus-background lighten($darkBgColor, 10%)
  --docsearch-hit-shadow none
  --docsearch-key-gradient linear-gradient(-26.5deg, #565862, #31353b)
  --docsearch-key-shadow inset 0 -2px 0 0 #282d55, inset 0 0 1px 1px #51577d, 0 2px 2px 0 rgba(3, 4, 9, 0.3)
  --docsearch-footer-shadow inset 0 1px 0 0 rgba(73, 76, 106, 0.5), 0 -4px 8px 0 rgba(0, 0, 0, 0.2)
  --docsearch-muted-color #7f8497

.DocSearch-Button
  display inline-flex
  justify-content space-between
  align-items center
  height 36px
  margin 0 1rem 0 0.25rem
  padding 0.5rem
  background var(--docsearch-searchbox-background)
  border-radius 40px
  color var(--docsearch-muted-color)
  font-weight 500
  user-select none
  outline none

  @media (max-width $MQMobile)
    margin-right 0

  &:active, &:focus, &:hover
    background var(--docsearch-searchbox-focus-background)
    box-shadow var(--docsearch-searchbox-shadow)
    color var(--dark-grey)

  &:hover .DocSearch-Search-Icon
    @media (max-width $MQMobile)
      color var(--accent-color)

.DocSearch-Button-Container
  .navbar &
    align-items center
    display flex

  .DocSearch-Search-Icon
    width 1rem
    height 1rem
    margin 0.1rem
    color #aaa
    stroke-width 3
    position relative

.DocSearch-Search-Icon
  stroke-width 1.6

.DocSearch-Button-Placeholder
  padding 0 12px 0 6px
  font-size 1rem

.DocSearch-Button-Keys
  .navbar &
    display flex

.DocSearch-Button-Key
  position relative
  top -1px
  width 20px
  height 18px
  margin-right 0.4em
  padding-bottom 2px
  border-radius 3px
  background var(--docsearch-key-gradient)
  box-shadow var(--docsearch-key-shadow)
  color var(--docsearch-muted-color)
  display flex
  justify-content center
  align-items center

  .navbar &
    display flex

.navbar
  @media (max-width $MQNarrow)
    .DocSearch-Button-Key, .DocSearch-Button-KeySeparator, .DocSearch-Button-Placeholder
      display none

.DocSearch--active
  overflow hidden !important

.DocSearch-Container
  position fixed
  top 0
  left 0
  z-index 200
  box-sizing border-box
  width 100vw
  height 100vh
  background-color var(--docsearch-container-background)

  *
    box-sizing border-box

  a
    text-decoration none

.DocSearch-Link
  margin 0
  padding 0
  border 0
  background none
  color var(--accent-color)
  font inherit
  appearance none
  cursor pointer

.DocSearch-Modal
  position relative
  max-width var(--docsearch-modal-width)
  margin 60px auto auto
  border-radius 6px
  background var(--bgcolor)
  box-shadow var(--docsearch-modal-shadow)
  flex-direction column

  @media (max-width $MQNarrow)
    width 100%
    max-width 100%
    height 100vh
    margin 0
    border-radius 0
    box-shadow none

.DocSearch-SearchBar
  display flex
  padding var(--docsearch-spacing) var(--docsearch-spacing) 0

.DocSearch-Form
  align-items center
  background var(--docsearch-searchbox-focus-background)
  border-radius 8px
  display flex
  height var(--docsearch-searchbox-height)
  padding 0 var(--docsearch-spacing)
  position relative
  width 100%

.DocSearch-Input
  width 80%
  height 100%
  padding 0 0 0 8px
  border 0
  background transparent
  color var(--text-color)
  font inherit
  font-size 1.2rem
  appearance none
  outline none
  flex 1

  &::placeholder
    color var(--docsearch-muted-color)
    opacity 1

  &::-webkit-search-cancel-button, &::-webkit-search-decoration, &::-webkit-search-results-button, &::-webkit-search-results-decoration
    display none

  @media (max-width $MQMobile)
    font-size 1rem

.DocSearch-LoadingIndicator, .DocSearch-MagnifierLabel, .DocSearch-Reset
  margin 0
  padding 0

.DocSearch-MagnifierLabel, .DocSearch-Reset
  align-items center
  color var(--accent-color)
  display flex
  justify-content center

.DocSearch-Container--Stalled .DocSearch-MagnifierLabel, .DocSearch-LoadingIndicator
  display none

.DocSearch-Container--Stalled .DocSearch-LoadingIndicator
  align-items center
  color var(--accent-color)
  display flex
  justify-content center

.DocSearch-Reset
  animation fade-in 0.1s ease-in forwards
  appearance none
  background none
  border 0
  border-radius 50%
  color var(--docsearch-icon-color)
  cursor pointer
  padding 2px
  right 0
  stroke-width var(--docsearch-icon-stroke-width)

  @media screen and (prefers-reduced-motion reduce)
    animation none
    appearance none
    background none
    border 0
    border-radius 50%
    color var(--docsearch-icon-color)
    cursor pointer
    right 0
    stroke-width var(--docsearch-icon-stroke-width)

  &[hidden]
    display none

  &:focus
    outline none

  &:hover
    color var(--accent-color)

.DocSearch-LoadingIndicator svg, .DocSearch-MagnifierLabel svg
  height 24px
  width 24px

.DocSearch-Cancel
  display none
  margin-left var(--docsearch-spacing)
  padding 0
  border 0
  background none
  color var(--accent-color)
  font-family Arial, Helvetica, sans-serif
  font-size 1em
  font-weight 500
  white-space nowrap
  user-select none
  appearance none
  cursor pointer
  flex none
  outline none
  overflow hidden

  @media (max-width $MQNarrow)
    display inline-block

.DocSearch-Dropdown
  max-height calc(var(--docsearch-modal-height) - var(--docsearch-searchbox-height) - var(--docsearch-spacing) - var(--docsearch-footer-height))
  min-height var(--docsearch-spacing)
  overflow-y auto
  overflow-y overlay
  padding 0 var(--docsearch-spacing)

  @media (max-width $MQNarrow)
    height 100%
    max-height unset

  & ul
    list-style none
    margin 0
    padding 0

.DocSearch-Label
  font-size 0.75em
  line-height 1.6em
  color var(--docsearch-muted-color)

.DocSearch-Help
  color var(--docsearch-muted-color)
  font-size 0.9em
  margin 0
  user-select none

.DocSearch-Title
  font-size 1.2em

.DocSearch-Logo
  a
    display flex

  svg
    margin-left 8px
    color var(--accent-color)

.DocSearch-Hits
  &:last-of-type
    margin-bottom 24px

  mark
    background none
    color var(--accent-color)

.DocSearch-HitsFooter
  margin-bottom var(--docsearch-spacing)
  padding var(--docsearch-spacing)
  color var(--docsearch-muted-color)
  font-size 0.85em
  display flex
  justify-content center

  a
    border-bottom 1px solid
    color inherit

.DocSearch-Hit
  position relative
  padding-bottom 4px
  border-radius 4px
  display flex

  a
    width 100%
    padding-left var(--docsearch-spacing)
    border-radius 4px
    background var(--bgcolor)
    box-shadow var(--docsearch-hit-shadow)
    display block

  &[aria-selected='true'] a
    background-color var(--accent-color)

  &[aria-selected='true'] mark
    text-decoration underline

.DocSearch-Hit--deleting
  opacity 0
  transition all 0.25s linear

  @media screen and (prefers-reduced-motion reduce)
    transition none

.DocSearch-Hit--favoriting
  transform scale(0)
  transform-origin top center
  transition all 0.25s linear
  transition-delay 0.25s

  @media screen and (prefers-reduced-motion reduce)
    transition none

.DocSearch-Hit-source
  position sticky
  top 0
  z-index 10
  margin 0 -4px
  padding 8px 4px 0
  background var(--bgcolor-light)
  color var(--accent-color)
  font-size 0.85em
  font-weight 600
  line-height 32px

.DocSearch-Hit-Tree
  width 24px
  height var(--docsearch-hit-height)
  color var(--docsearch-muted-color)
  opacity 0.5
  stroke-width var(--docsearch-icon-stroke-width)

  @media (max-width $MQNarrow)
    display none

.DocSearch-Hit-Container
  height var(--docsearch-hit-height)
  padding 0 var(--docsearch-spacing) 0 0
  color var(--docsearch-hit-color)
  display flex
  flex-direction row
  align-items center

.DocSearch-Hit-icon
  width 20px
  height 20px
  color var(--docsearch-muted-color)
  stroke-width var(--docsearch-icon-stroke-width)

.DocSearch-Hit-action
  width 22px
  height 22px
  color var(--docsearch-muted-color)
  stroke-width var(--docsearch-icon-stroke-width)
  display flex
  align-items center

  svg
    display block
    width 18px
    height 18px

  & + &
    margin-left 6px

.DocSearch-Hit-action-button
  padding 2px
  border 0
  border-radius 50%
  background none
  color inherit
  cursor pointer
  appearance none

  &:focus, &:hover
    background rgba(0, 0, 0, 0.2)
    transition background-color 0.1s ease-in

    @media screen and (prefers-reduced-motion reduce)
      background rgba(0, 0, 0, 0.2)
      transition none

    path
      fill #fff

svg.DocSearch-Hit-Select-Icon
  display none

.DocSearch-Hit[aria-selected='true'] .DocSearch-Hit-Select-Icon
  display block

.DocSearch-Hit-content-wrapper
  width 80%
  position relative
  margin 0 8px
  font-weight 500
  line-height 1.2em
  text-overflow ellipsis
  white-space nowrap
  display flex
  flex 1 1 auto
  flex-direction column
  justify-content center
  overflow-x hidden

.DocSearch-Hit-title
  font-size 0.9em

.DocSearch-Hit-path
  color var(--docsearch-muted-color)
  font-size 0.75em

.DocSearch-Hit[aria-selected='true']
  .DocSearch-Hit-action, .DocSearch-Hit-icon, .DocSearch-Hit-path, .DocSearch-Hit-text, .DocSearch-Hit-title, .DocSearch-Hit-Tree, mark
    color var(--bgcolor) !important

.DocSearch-ErrorScreen, .DocSearch-NoResults, .DocSearch-StartScreen
  width 80%
  margin 0 auto
  padding 36px 0
  font-size 0.9em
  text-align center

.DocSearch-Screen-Icon
  padding-bottom 12px
  color var(--docsearch-muted-color)

.DocSearch-NoResults-Prefill-List
  display inline-block
  padding-bottom 24px
  text-align left

  ul
    display inline-block
    padding 8px 0 0

  li
    list-style-position inside
    list-style-type '» '

.DocSearch-Prefill
  appearance none
  background none
  border 0
  border-radius 1em
  color var(--accent-color)
  cursor pointer
  display inline-block
  font-size 1em
  font-weight 700
  padding 0

  &:focus, &:hover
    outline none
    text-decoration underline

.DocSearch-Footer
  position relative
  z-index 300
  width 100%
  height var(--docsearch-footer-height)
  padding 0 var(--docsearch-spacing)
  border-radius 0 0 8px 8px
  box-shadow var(--docsearch-footer-shadow)
  background var(--bgcolor)
  display flex
  flex-direction row-reverse
  flex-shrink 0
  align-items center
  justify-content space-between
  user-select none

  @media (max-width $MQNarrow)
    position absolute
    bottom 0
    border-radius 0

.DocSearch-Commands
  margin 0
  padding 0
  color var(--docsearch-muted-color)
  display flex
  list-style none

  @media (max-width $MQNarrow)
    display none

  li
    display flex
    align-items center

    &:not(:last-of-type)
      margin-right 0.8em

.DocSearch-Commands-Key
  width 20px
  height 18px
  margin-right 0.4em
  padding-bottom 2px
  border-radius 2px
  background var(--docsearch-key-gradient)
  box-shadow var(--docsearch-key-shadow)
  display flex
  justify-content center
  align-items center
</style>
