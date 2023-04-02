<template>
  <transition
    name="drop"
    appear
    @appear="setStyle"
    @after-appear="unsetStyle"
    @enter="setStyle"
    @after-enter="unsetStyle"
    @before-leave="setStyle"
  >
    <slot />
  </transition>
</template>

<script src="./MyTransition" />

<style lang="stylus">
.drop-enter, .drop-leave-to
  opacity 0
  transform translateY(-20px)
</style>
