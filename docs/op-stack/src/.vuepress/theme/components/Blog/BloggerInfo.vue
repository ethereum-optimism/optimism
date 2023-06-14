<template>
  <div class="blogger-info" vocab="https://schema.org/" typeof="Person">
    <div
      class="blogger"
      :class="{ hasIntro }"
      :[hintAttr]="hasIntro ? i18n.intro : ''"
      :data-balloon-pos="hasIntro ? 'down' : ''"
      role="navigation"
      @click="jumpIntro"
    >
      <img
        v-if="bloggerAvatar"
        class="avatar"
        :class="{ round: blogConfig.roundAvatar !== false }"
        property="image"
        alt="Blogger Avatar"
        :src="$withBase(bloggerAvatar)"
      />
      <div
        v-if="bloggerName"
        class="name"
        property="name"
        v-text="bloggerName"
      />
      <meta
        v-if="hasIntro"
        property="url"
        :content="$withBase(blogConfig.intro)"
      />
    </div>
    <div class="num-wrapper">
      <div @click="navigate('/article/')">
        <div class="num">{{ articleNumber }}</div>
        <div>{{ i18n.article }}</div>
      </div>
      <div @click="navigate('/category/')">
        <div class="num">{{ $category.list.length }}</div>
        <div>{{ i18n.category }}</div>
      </div>
      <div @click="navigate('/tag/')">
        <div class="num">{{ $tag.list.length }}</div>
        <div>{{ i18n.tag }}</div>
      </div>
      <div @click="navigate('/timeline/')">
        <div class="num">{{ $timelineItems.length }}</div>
        <div>{{ i18n.timeline }}</div>
      </div>
    </div>
    <MediaLinks />
  </div>
</template>

<script src="./BloggerInfo" />

<style lang="stylus">
.blogger-info
  .page &
    background var(--bgcolor)

  .blogger
    padding 8px 0
    text-align center

    &.hasIntro
      cursor pointer

    .avatar
      width 128px
      height 128px
      margin 0 auto

      &.round
        border-radius 50%

    .name
      margin 16px auto
      font-size 22px

  .num-wrapper
    display flex
    margin 0 auto 16px
    width 80%

    > div
      width 25%
      text-align center
      font-size 13px
      cursor pointer

      &:hover
        color var(--accent-color)

      .num
        position relative
        margin-bottom 8px
        font-weight 600
        font-size 20px
</style>
