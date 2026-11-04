<script setup>
import { toasts } from '../utils/toast';
</script>

<template>
  <div class="toast-container">
    <transition-group name="toast-anim" tag="div" class="toast-wrapper">
      <div 
        v-for="toast in toasts" 
        :key="toast.id" 
        class="toast"
        :class="`toast-${toast.type}`"
      >
        {{ toast.message }}
      </div>
    </transition-group>
  </div>
</template>

<style scoped>
.toast-container {
  position: fixed;
  top: 40px;
  left: 50%;
  transform: translateX(-50%);
  z-index: 9999;
  display: flex;
  flex-direction: column;
  align-items: center;
  pointer-events: none; /* Let clicks pass through */
}

.toast-wrapper {
  display: flex;
  flex-direction: column-reverse; /* Newest at top */
  align-items: center;
  gap: 12px;
}

.toast {
  background-color: var(--text-color);
  color: var(--bg-color);
  padding: 12px 24px;
  border-radius: 8px;
  font-size: 15px;
  font-weight: 600;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.15);
  pointer-events: auto;
  max-width: 80vw;
  text-align: center;
  word-break: break-word;
}

/* Transitions for fading in, sliding up and disappearing */
.toast-anim-enter-active,
.toast-anim-leave-active {
  transition: all 0.4s cubic-bezier(0.2, 0.8, 0.2, 1);
}

.toast-anim-enter-from {
  opacity: 0;
  transform: translateY(30px);
}

.toast-anim-leave-to {
  opacity: 0;
  transform: translateY(-30px);
}
</style>
