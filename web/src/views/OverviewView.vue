<script setup lang="ts">
</script>

<template>
  <div class="grid">
    <div class="card stat">
      <h3>欢迎使用 Wechat2RSS</h3>
      <p>从左侧导航进入公众号、任务、微信会话菜单即可开始配置。</p>
      <p class="tips">当前为单用户部署，所有配置仅在本实例生效。</p>
    </div>
  </div>
</template>

<style scoped>
.grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  gap: 1rem;
}

.stat h3 {
  margin-top: 0;
}

.tips {
  color: #64748b;
}
</style>
