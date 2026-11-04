<script setup>
import { confirmState, closeConfirm } from '../utils/confirm';
</script>

<template>
  <div v-if="confirmState.isOpen" class="modal-overlay" @click="closeConfirm(false)">
    <div class="confirm-modal" @click.stop>
      <div class="confirm-message">{{ confirmState.message }}</div>
      <div class="confirm-actions">
        <button class="cancel-btn" @click="closeConfirm(false)">取消</button>
        <button class="confirm-btn" @click="closeConfirm(true)">确定</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background-color: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 10000;
  animation: fadeIn 0.2s ease-out;
}
.confirm-modal {
  background-color: var(--bg-color);
  color: var(--text-color);
  padding: 24px;
  border-radius: 12px;
  width: 90%;
  max-width: 400px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
  text-align: center;
  animation: scaleIn 0.2s ease-out;
}
.confirm-message {
  font-size: 1.1rem;
  margin-bottom: 24px;
  line-height: 1.5;
}
.confirm-actions {
  display: flex;
  justify-content: space-around;
  gap: 16px;
}
.cancel-btn, .confirm-btn {
  flex: 1;
  padding: 10px 0;
  border: none;
  border-radius: 8px;
  font-size: 1rem;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.2s;
}
.cancel-btn {
  background-color: transparent;
  border: 1px solid var(--border-color);
  color: var(--text-color);
}
.confirm-btn {
  background-color: #ff4d4f;
  color: white;
}
.cancel-btn:hover, .confirm-btn:hover {
  opacity: 0.8;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}
@keyframes scaleIn {
  from { transform: scale(0.9); opacity: 0; }
  to { transform: scale(1); opacity: 1; }
}
</style>
