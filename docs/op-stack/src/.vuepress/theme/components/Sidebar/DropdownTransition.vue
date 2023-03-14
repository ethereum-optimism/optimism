<template>
  <transition
    name="dropdown"
    @enter="setHeight"
    @after-enter="unsetHeight"
    @before-leave="setHeight"
  >
    <slot />
  </transition>
</template>

<script src="./DropdownTransition" />

<style lang="stylus">
.dropdown-enter, .dropdown-leave-to
  height 0 !important
</style>
