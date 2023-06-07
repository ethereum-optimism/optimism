<template>
  <div
    class="theme-container"
    :class="pageClasses"
    @touchstart="onTouchStart"
    @touchend="onTouchEnd"
  >
    <Password v-if="isGlobalEncrypted" @password-verify="checkGlobalPassword" />
    <!-- Content -->
    <template v-else>
      <Navbar v-if="enableNavbar" @toggle-sidebar="toggleSidebar">
        <template #start>
          <slot name="navbar-start" />
        </template>
        <template #center>
          <slot name="navbar-center" />
        </template>
        <template #end>
          <slot name="navbar-end" />
        </template>
      </Navbar>

      <div class="sidebar-mask" @click="toggleSidebar(false)" />

      <Sidebar :items="sidebarItems" @toggle-sidebar="toggleSidebar">
        <template #top>
          <slot name="sidebar-top" />
        </template>
        <template #center>
          <slot name="sidebar-center" />
        </template>
        <template #bottom>
          <slot name="sidebar-bottom" />
        </template>
      </Sidebar>

      <slot :sidebar-items="sidebarItems" :headers="headers" />

      <PageFooter :key="$route.path" />
    </template>
  </div>
</template>

<script src="./Common" />

<style lang="stylus">
.theme-container
  min-height 100vh

.sidebar-mask
  position fixed
  z-index 9
  top 0
  left 0
  width 100vw
  height 100vh
  display none

  .theme-container.sidebar-open &
    display block
</style>
