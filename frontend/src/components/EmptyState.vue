<template>
  <div class="empty-state">
    <div class="empty-state__art">
      <svg viewBox="0 0 120 96" aria-hidden="true">
        <defs>
          <linearGradient id="nd-empty-grad" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%" stop-color="#2B7FFF" stop-opacity="0.85" />
            <stop offset="100%" stop-color="#22D3EE" stop-opacity="0.55" />
          </linearGradient>
        </defs>
        <rect x="14" y="22" width="92" height="58" rx="14" fill="url(#nd-empty-grad)" opacity="0.18" />
        <rect x="26" y="34" width="68" height="34" rx="10" fill="url(#nd-empty-grad)" opacity="0.28" />
        <circle cx="60" cy="51" r="10" fill="#fff" opacity="0.5" />
        <path d="M60 45v12M54 51h12" stroke="#1F2329" stroke-width="2.4" stroke-linecap="round" opacity="0.55" />
      </svg>
    </div>
    <p class="empty-state__title">{{ title }}</p>
    <p v-if="description" class="empty-state__desc">{{ description }}</p>
    <div v-if="$slots.default" class="empty-state__action">
      <slot />
    </div>
  </div>
</template>

<script setup>
defineProps({
  title: { type: String, default: '这里空空如也' },
  description: { type: String, default: '' }
})
</script>

<style scoped>
.empty-state {
  padding: 56px 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
}
.empty-state__art {
  width: 128px;
  margin-bottom: 10px;
  filter: drop-shadow(0 12px 28px rgba(43, 127, 255, 0.18));
}
.empty-state__art svg {
  width: 100%;
  height: auto;
}
.empty-state__title {
  font-size: 15px;
  font-weight: 600;
}
.empty-state__desc {
  margin-top: 6px;
  font-size: 13px;
  color: var(--nd-text-secondary);
  max-width: 420px;
  line-height: 1.7;
}
.empty-state__action {
  margin-top: 16px;
}
</style>
