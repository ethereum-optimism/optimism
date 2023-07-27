<template>
  <main class="page">

    <slot name="top" />

    <BreadCrumb :key="$route.path" />



    <PageInfo :key="$route.path" />

    <MyTransition v-if="pagePassword && !pageDescrypted" :delay="0.08" :disable="true">
      <Password
        :key="$route.path"
        :page="true"
        @password-verify="password = $event"
      />
    </MyTransition>

    <MyTransition v-else-if="isPathEncrypted" :delay="0.08" :disable="true">
      <Password
        :key="$route.path"
        :page="true"
        @password-verify="checkPathPassword"
      />
    </MyTransition>

    <template v-else>
      <MyTransition :delay="0.12" :disable="true">
        <Anchor :key="$route.path" />
      </MyTransition>

      <slot v-if="!pagePassword || pageDescrypted" name="content-top" />

      <MyTransition v-show="!pagePassword || pageDescrypted" :delay="0.08" :disable="true">
        <Content :key="$route.path" class="theme-default-content" />
      </MyTransition>

      <slot v-if="!pagePassword || pageDescrypted" name="content-bottom" />

      <MyTransition :delay="0.12" :disable="true">
        <PageMeta :key="$route.path" />
      </MyTransition>

      <MyTransition :delay="0.14" :disable="true">
        <PageNav :key="$route.path" v-bind="{ sidebarItems }" />
      </MyTransition>

      <MyTransition :delay="0.16" :disable="true">
        <Comment :key="$route.path" />
      </MyTransition>
    </template>

    <slot name="bottom" />

    <!-- Google tag (gtag.js) -->
    <!-- put here because the plugin didn't work -->
    <script async src="https://www.googletagmanager.com/gtag/js?id=G-9KLVB8X0ME">      
    </script>

    <script>
      window.dataLayer = window.dataLayer || [];
      function gtag(){dataLayer.push(arguments);}
      gtag('js', new Date());
      gtag('config', 'G-9KLVB8X0ME');
    </script>
  </main>
</template>

<script src="./Page" />

<style lang="stylus">
.page
  display block
  box-sizing border-box
  min-height 100vh
  padding-left $sidebarWidth
  padding-bottom 2rem
  background var(--bgcolor)

  @media (max-width $MQMobile)
    min-height 100vh

  // narrow desktop / iPad
  @media (max-width $MQNarrow)
    padding-left $mobileSidebarWidth

  // wide mobile
  @media (max-width $MQMobile)
    padding-left 0

  @media (min-width $MQMobile)
    .theme-container:not(.has-sidebar) &
      padding-left 0

  @media (min-width $MQWide)
    .has-anchor &:not(.blog)
      padding-right 16rem
</style>
