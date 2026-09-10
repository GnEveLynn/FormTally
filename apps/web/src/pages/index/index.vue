<template>
  <main class="start-page">
    <h1>FormTally</h1>
    <p>记录饮食，了解每天的营养摄入</p>
  </main>
</template>

<style scoped>
.start-page {
  min-height: 100svh;
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  gap: var(--ft-space-sm);
  padding: var(--ft-space-lg);
  text-align: center;
}

h1 {
  margin: 0;
  font-size: var(--ft-font-size-title);
  line-height: 1.2;
}

p {
  margin: 0;
  color: var(--ft-color-text-muted);
  font-size: var(--ft-font-size-body);
  line-height: 1.6;
}
</style>
