<template>
  <div class="auth">
    <div class="auth__glow auth__glow--one" />
    <div class="auth__glow auth__glow--two" />

    <section class="auth__brand">
      <div class="nd-logo auth__logo">
        <span class="nd-logo__mark">
          <el-icon><Cloudy /></el-icon>
        </span>
        <span>NovaDrive</span>
      </div>

      <h1 class="auth__title">
        星云盘
        <span class="text-gradient">v2</span>
      </h1>
      <p class="auth__subtitle">
        自己的私有云盘。分片上传、秒传、在线预览与链接分享，全部装进一个二进制文件。
      </p>

      <ul class="auth__features">
        <li v-for="feat in features" :key="feat.title">
          <span class="auth__feature-icon">
            <el-icon><component :is="feat.icon" /></el-icon>
          </span>
          <div>
            <p class="auth__feature-title">{{ feat.title }}</p>
            <p class="auth__feature-desc">{{ feat.desc }}</p>
          </div>
        </li>
      </ul>

      <p class="auth__foot text-muted">NovaDrive · 星云盘 · 基于 Go + Vue 3 构建</p>
    </section>

    <section class="auth__panel glass-card">
      <slot />
    </section>
  </div>
</template>

<script setup>
const features = [
  { icon: 'UploadFilled', title: '分片上传 + 秒传', desc: '大文件断点续传，相同内容只存一份' },
  { icon: 'View', title: '五类在线预览', desc: '图片 / 视频 / 音频 / PDF / 文本开箱可用' },
  { icon: 'Share', title: '链接与提取码分享', desc: '二维码、有效期与下载次数限制' }
]
</script>

<style scoped>
.auth {
  min-height: 100vh;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(380px, 440px);
  align-items: center;
  gap: 56px;
  padding: 48px clamp(24px, 7vw, 120px);
  position: relative;
  overflow: hidden;
}
.auth__glow {
  position: absolute;
  border-radius: 50%;
  filter: blur(90px);
  opacity: 0.5;
  pointer-events: none;
}
.auth__glow--one {
  width: 420px;
  height: 420px;
  background: #2b7fff;
  top: -120px;
  left: -80px;
  opacity: 0.22;
}
.auth__glow--two {
  width: 360px;
  height: 360px;
  background: #22d3ee;
  bottom: -140px;
  right: 18%;
  opacity: 0.32;
}
.auth__brand {
  position: relative;
  z-index: 1;
  max-width: 540px;
}
.auth__logo {
  margin-bottom: 28px;
}
.auth__title {
  font-size: clamp(30px, 3.6vw, 44px);
  line-height: 1.15;
  font-weight: 700;
  letter-spacing: -0.5px;
}
.auth__subtitle {
  margin-top: 14px;
  font-size: 14px;
  line-height: 1.8;
  color: var(--nd-text-secondary);
  max-width: 440px;
}
.auth__features {
  list-style: none;
  margin: 32px 0 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.auth__features li {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}
.auth__feature-icon {
  width: 34px;
  height: 34px;
  border-radius: 11px;
  flex: none;
  display: grid;
  place-items: center;
  background: rgba(43, 127, 255, 0.12);
  border: 1px solid var(--nd-border);
  color: var(--nd-primary);
  font-size: 16px;
}
.auth__feature-title {
  font-size: 13.5px;
  font-weight: 600;
}
.auth__feature-desc {
  font-size: 12.5px;
  color: var(--nd-text-secondary);
  margin-top: 2px;
}
.auth__foot {
  margin-top: 36px;
  font-size: 11.5px;
}
.auth__panel {
  position: relative;
  z-index: 1;
  width: 100%;
  padding: 32px 28px;
  background: var(--nd-surface);
}
@media (max-width: 1023px) {
  .auth {
    grid-template-columns: minmax(0, 1fr);
    gap: 32px;
    padding: 40px 20px 48px;
  }
  .auth__brand {
    text-align: center;
    margin: 0 auto;
  }
  .auth__logo {
    justify-content: center;
  }
  .auth__subtitle {
    margin-left: auto;
    margin-right: auto;
  }
  .auth__features {
    display: none;
  }
  .auth__foot {
    margin-top: 18px;
  }
  .auth__panel {
    max-width: 440px;
    margin: 0 auto;
  }
}
</style>
