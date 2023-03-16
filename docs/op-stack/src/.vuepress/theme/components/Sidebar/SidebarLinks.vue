<template>
  <ul v-if="items.length" class="sidebar-links">
    <li v-for="(item, index) in items" :key="index">
      <SidebarGroup
        v-if="item.type === 'group'"
        :item="item"
        :open="index === openGroupIndex"
        :depth="depth"
        @toggle="toggleGroup(index)"
      />
      <SidebarLink v-else :item="item" />
    </li>
  </ul>
</template>

<script src="./SidebarLinks" />
