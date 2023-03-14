<template>
  <footer class="page-meta">
    <div class="footer-box">
      <div class="footer-box-area">
        <span class="footer-section-header">About this page</span>
        <ul>
          <li>Updated: {{ updateTime.slice(0, updateTime.length - 6) }}</li>
        </ul>
      </div>

      <div class="footer-box-area">
        <span class="footer-section-header">Contribute</span>
        <ul>
          <li>
            <a :href="editLink" target="_blank" rel="noopener noreferrer">
              <i class="far fa-pencil"></i> Edit this page
            </a>
          </li>
          <li>
            <a href="https://github.com/ethereum-optimism/optimism/contribute" target="_blank" rel="noopener noreferrer">
              <i class="far fa-hands-helping"></i> Contribute to Optimism
            </a>
          </li>
        </ul>
      </div>

      <div class="footer-box-area">
        <span class="footer-section-header">Still need help?</span>
        <ul>
          <li>
            <a href="https://discord.optimism.io" target="_blank" rel="noopener noreferrer">
              <i class="fab fa-discord"></i> Discord community
            </a>
          </li>
        </ul>
      </div>
    </div>
  </footer>
</template>

<script src="./PageMeta" />

<style lang="stylus">
@require '~@mr-hope/vuepress-shared/styles/wrapper'

.page-meta
  @extend $wrapper
  padding-top 12px
  padding-bottom 12px
  font-family Arial, Helvetica, sans-serif
  overflow auto

  .meta-item
    .label
      font-weight 500
      color var(--text-color-l25)

    .info
      font-weight 400
      color var(--dark-grey)

  .edit-link
    display inline-block
    font-size 14px

    .icon
      position relative
      bottom -0.125em
      width 1em
      height 1em
      color var(--accent-color)

    @media (max-width $MQMobile)
      margin-bottom 8px

    a
      color var(--accent-color-l10)

  .update-time
    float right
    font-size 14px

    @media (max-width $MQMobile)
      float none
      font-size 13px
      text-align left

  .contributors
    font-size 14px
    text-align right

    @media (max-width $MQMobile)
      font-size 13px
      text-align left

  .footer-box
    display flex
    flex-direction row
    justify-content space-between
    background-color #F1F4F9
    padding 32px
    padding-left: 40px;
    padding-right: 40px;
    border-radius 16px

    @media (max-width $MQNarrow)
      flex-direction column

    .footer-box-area
      @media (max-width $MQNarrow)
        margin-bottom 32px

      span.footer-section-header
        font-family 'Open Sans', sans-serif
        font-weight 600
        font-size 14px
        line-height 20px

      ul
        list-style-type none
        padding-left 0
        font-size 14px
        line-height 20px
        margin-top 10px
        margin-bottom 0px
        color #68778D

        li
          margin-top 15px
          margin-bottom 5px

      a
        color #68778D
        font-family 'Open Sans', sans-serif

        &:hover
          color #FF0420

      i
        font-size 14px
        width 20px
        margin-right 3px
        text-align center
</style>
