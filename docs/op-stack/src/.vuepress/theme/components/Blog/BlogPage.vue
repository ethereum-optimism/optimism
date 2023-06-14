<template>
  <main class="blog-page">
    <MyTransition>
      <component :is="componentName" v-if="componentName" />
    </MyTransition>

    <MyTransition :delay="0.24">
      <ArticleList v-if="showArticles" :key="$route.path" />
    </MyTransition>
  </main>
</template>

<script src="./BlogPage" />

<style lang="stylus">
.blog-page
  max-width 780px
  flex 1

  @media (min-width $MQMobile)
    margin 0 15px

  .article-title
    font-size 1.8rem
    margin 10px 15px
</style>
