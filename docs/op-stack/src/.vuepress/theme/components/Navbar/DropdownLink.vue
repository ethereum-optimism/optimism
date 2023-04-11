<template>
  <div class="dropdown-wrapper" :class="{ open }">
    <button
      class="dropdown-title"
      type="button"
      :aria-label="dropdownAriaLabel"
      @click="handleDropdown"
    >
      <slot name="title">
        <span class="title">
          <i v-if="item.icon" :class="`iconfont ${iconPrefix}${item.icon}`" />
          {{ item.text }}
        </span>
      </slot>
      <span class="arrow" />
    </button>

    <ul class="nav-dropdown">
      <li
        v-for="(child, index) in item.items"
        :key="child.link || index"
        class="dropdown-item"
      >
        <template v-if="child.type === 'links'">
          <h4 class="dropdown-subtitle">
            <NavLink
              v-if="child.link"
              :item="child"
              @focusout="
                isLastItemOfArray(child, item.children) &&
                  child.children.length === 0 &&
                  setOpen(false)
              "
            />

            <span v-else>{{ child.text }}</span>
          </h4>

          <ul class="dropdown-subitem-wrapper">
            <li
              v-for="grandchild in child.items"
              :key="grandchild.link"
              class="dropdown-subitem"
            >
              <NavLink
                :item="grandchild"
                @focusout="
                  isLastItemOfArray(grandchild, child.items) &&
                    isLastItemOfArray(child, item.items) &&
                    setOpen(false)
                "
              />
            </li>
          </ul>
        </template>

        <NavLink
          v-else
          :item="child"
          @focusout="isLastItemOfArray(child, item.items) && setOpen(false)"
        />
      </li>
    </ul>
  </div>
</template>

<script src="./DropdownLink" />

<style lang="stylus">
@require '~@mr-hope/vuepress-shared/styles/arrow'
@require '~@mr-hope/vuepress-shared/styles/reset'

.dropdown-wrapper
  height 1.8rem
  cursor pointer

  &:not(:hover)
    .arrow
      transform rotate(-180deg)

  &:hover, &.open
    .nav-dropdown
      z-index 2
      transform scale(1)
      visibility visible
      opacity 1

  .dropdown-title
    button()
    cursor inherit
    padding inherit
    color var(--dark-grey)
    font-family inherit
    font-size 0.9rem
    font-weight 500
    line-height 1.4rem

    &::after
      border-left 5px solid var(--accent-color)

    &:hover
      border-color transparent

    .arrow
      arrow()
      font-size 1.2em

  .nav-dropdown
    box-sizing border-box
    position absolute
    top 100%
    right 0
    max-height 100vh - $navbarHeight
    margin 0
    padding 0.6rem 0
    border 1px solid var(--grey14)
    border-radius 0.25rem
    background var(--bgcolor)
    box-shadow 2px 2px 10px var(--card-shadow-color)
    text-align left
    white-space nowrap
    overflow-y auto
    transform scale(0.8)
    opacity 0
    visibility hidden
    transition all 0.18s ease-out

  .dropdown-item
    color inherit
    line-height 1.7rem

    h4
      margin 0
      padding 0.75rem 1rem 0.25rem 0.75rem
      border-top 1px solid var(--grey14)
      color var(--dark-grey)
      font-size 0.9rem

      .nav-link
        padding 0

        &:before
          display none

    &:first-child h4
      padding-top 0
      border-top 0

    .nav-link
      display block
      position relative
      margin-bottom 0
      padding 0 1.5rem 0 1.25rem
      border-bottom none
      color var(--dark-grey)
      font-weight 400
      line-height 1.7rem

      &:hover
        color var(--accent-color)

      &.active
        color var(--accent-color)

        &::before
          content ''
          position absolute
          top calc(50% - 3px)
          left 9px
          width 0
          height 0
          border-top 3px solid transparent
          border-left 5px solid var(--accent-color)
          border-bottom 3px solid transparent

    .dropdown-subitem-wrapper
      padding 0
      list-style none

    .dropdown-subitem
      font-size 0.9em
</style>
