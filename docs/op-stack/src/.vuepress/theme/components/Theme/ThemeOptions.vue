<template>
  <div class="theme-options">
    <ul v-if="themeColorEnabled" class="themecolor-select">
      <label for="themecolor-select" v-text="`${text.themeColor}:`" />
      <li>
        <span class="default-theme" @click="setTheme()" />
      </li>
      <li v-for="color in themeColor.list" :key="color">
        <span
          :style="{ background: themeColor.picker[color] }"
          @click="setTheme(color)"
        />
      </li>
    </ul>
    <div v-if="switchEnabled" class="darkmode-toggle">
      <label class="desc" for="darkmode-toggle" v-text="`${text.themeMode}:`" />
      <DarkmodeSwitch />
      <ScreenFull />
    </div>
  </div>
</template>

<script src="./ThemeOptions" />

<style lang="stylus">
.theme-options
  font-size 14px

  .themecolor-select
    display flex
    justify-content space-around

    label
      padding-right 8px

    li
      &:first-child
        margin-right 8px

      span
        vertical-align middle
        width 15px
        height 15px
        margin 0 2px
        border-radius 2px

        &.default-theme
          background $accentColor // must be fixed to the original accent-color

  .darkmode-toggle
    display flex
    align-items center
    margin-top 8px

    .desc
      padding-right 8px
      line-height 1.5

    .full-screen, .cancel-full-screen
      margin-left 0.5em
</style>
