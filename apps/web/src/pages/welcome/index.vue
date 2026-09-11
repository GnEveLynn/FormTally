<template>
  <main class="welcome-page">
    <div class="brand">FT</div>
    <p class="eyebrow">FORMTALLY</p>
    <h1>看见每一餐，了解每一天</h1>
    <p class="description">拍下饮食，核对 AI 估算，轻松掌握每日热量与营养摄入。</p>
    <RouterLink class="primary-action" to="/login">开始使用</RouterLink>
    <p class="privacy-note">你的身体资料与饮食记录默认仅自己可见</p>
  </main>
</template>

<style scoped>
.welcome-page {
  min-height: 100svh;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: 1rem;
  padding: 2rem 1.5rem;
  text-align: center;
}

.brand {
  display: grid;
  place-items: center;
  width: 4.5rem;
  height: 4.5rem;
  border-radius: 1.5rem;
  color: white;
  background: var(--ft-color-text);
  font-size: 1.35rem;
  font-weight: 800;
  letter-spacing: .08em;
}

.eyebrow { margin: .5rem 0 0; color: var(--ft-color-text-muted); font-size: .75rem; letter-spacing: .2em; }
h1 { max-width: 18rem; margin: 0; font-size: clamp(2rem, 9vw, 3rem); line-height: 1.12; }
.description { max-width: 22rem; margin: 0; color: var(--ft-color-text-muted); line-height: 1.7; }
.primary-action {
  display: grid;
  place-items: center;
  width: min(100%, 22rem);
  min-height: 3.25rem;
  margin-top: 1.5rem;
  border-radius: 999px;
  color: white;
  background: var(--ft-color-text);
  text-decoration: none;
  font-weight: 700;
}
.privacy-note { margin: .5rem 0 0; color: var(--ft-color-text-muted); font-size: .8rem; }
</style>
