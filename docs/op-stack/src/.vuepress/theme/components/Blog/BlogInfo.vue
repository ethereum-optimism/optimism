<template>
  <aside class="blog-info-wrapper">
    <MyTransition>
      <BloggerInfo />
    </MyTransition>

    <MyTransition :delay="0.04">
      <BlogInfoList />
    </MyTransition>
  </aside>
</template>

<script src="./BlogInfo" />

<style lang="stylus">
.blog-info-wrapper
  .sidebar &
    .blogger-info
      display none

  .page &
    position sticky
    box-sizing border-box
    top ($navbarHeight + 1rem)
    flex 0 0 300px
    height auto
    margin-bottom 12px
    transition all 0.3s

    @media (max-width $MQMobile)
      display none

    .blogger-info
      margin-bottom 16px
      padding 8px 0
      border-radius 8px
      box-shadow 0 1px 3px 0 var(--card-shadow-color)

      &:hover
        box-shadow 0 2px 6px 0 var(--card-shadow-color)
</style>
