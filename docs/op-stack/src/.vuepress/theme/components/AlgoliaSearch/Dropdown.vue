<template>
  <form
    id="search-form"
    class="algolia-search-wrapper search-box"
    role="search"
  >
    <label class="sr-only" for="algolia-search-input">Algolia search</label>
    <input
      id="algolia-search-input"
      class="search-query"
      :placeholder="placeholder"
    />
  </form>
</template>

<script src="./Dropdown" />

<style lang="stylus">
.algolia-search-wrapper
  & > span
    vertical-align middle

  .algolia-autocomplete
    line-height normal

    .ds-dropdown-menu
      min-width 515px !important
      margin 6px 0 0
      padding 4px
      border 1px solid var(--light-grey)
      border-radius 4px
      background var(--bgcolor)
      font-size 16px
      text-align left

      @media (max-width $MQMobile)
        min-width calc(100vw - 4rem) !important
        max-width calc(100vw - 4rem) !important

      &:before
        border-color var(--light-grey)

      [class*=ds-dataset-]
        padding 0
        border none
        background var(--bgcolor)

      .ds-suggestions
        margin-top 0

      .ds-suggestion
        border-bottom 1px solid var(--border-color)

    .algolia-docsearch-suggestion--highlight
      color var(--accent-color)

    .algolia-docsearch-suggestion
      padding 0
      border-color var(--border-color)
      background var(--bgcolor)
      color var(--text-color)

      .algolia-docsearch-suggestion--category-header
        padding 5px 10px
        margin-top 0
        background var(--accent-color)
        color var(--white)
        font-weight 600

        .algolia-docsearch-suggestion--highlight
          background rgba(255, 255, 255, 0.6)

      .algolia-docsearch-suggestion--wrapper
        padding 0

        @media (max-width $MQMobile)
          padding 5px 7px 5px 5px !important

      .algolia-docsearch-suggestion--title
        margin-bottom 0
        color var(--text-color)
        font-weight 600

      .algolia-docsearch-suggestion--subcategory-column
        vertical-align top
        padding 5px 7px 5px 5px
        border-color var(--border-color)
        background var(--bgcolor)
        color var(--text-color)

        @media (min-width $MQMobile)
          display table-cell
          float none
          width 150px
          min-width 150px

        @media (max-width $MQMobile)
          padding 0 !important
          background white !important

        &:after
          display none

      .algolia-docsearch-suggestion--subcategory-column-text
        color #555

        &:after
          @media (max-width $MQMobile)
            display inline-block
            vertical-align middle
            content ' > '
            width 5px
            margin -3px 3px 0
            font-size 10px
            line-height 14.4px

      .algolia-docsearch-suggestion--content
        @media (min-width $MQMobile)
          display table-cell
          float none
          vertical-align top
          width 100%

    .algolia-docsearch-footer
      border-color var(--border-color)

    .ds-cursor .algolia-docsearch-suggestion--content
      background var(--grey3)
      color var(--text-color)
</style>
