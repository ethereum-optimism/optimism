<template>
  <!-- eslint-disable vue/no-deprecated-v-on-native-modifier -->
  <RouterLink
    v-if="isInternal"
    class="nav-link"
    :class="{ active }"
    :to="link"
    @focusout.native="focusoutAction"
  >
    <i v-if="item.icon" :class="`iconfont ${item.iconPrefix || iconPrefix}${item.icon} ${item.iconClass}`" />
    {{ item.text }}
  </RouterLink>
  <a
    v-else
    class="nav-link external"
    :href="link"
    :target="target"
    :rel="rel"
    @focusout="focusoutAction"
  >
    <i v-if="item.icon" :class="`iconfont ${item.iconPrefix || iconPrefix}${item.icon} ${item.iconClass}`" />
    {{ item.text }}
    <OutboundLink v-if="isBlankTarget" />
  </a>
</template>

<script src="./NavLink" />

<style lang="stylus">
.nav-link
  line-height 1.4rem

  .navbar &
    color var(--dark-grey)

    &.active
      color var(--accent-color)

  .sidebar &
    color var(--text-color)

    &:hover, &.active
      color var(--accent-color)
</style>
