import { ref } from 'vue';

export const toasts = ref([]);

let idCounter = 0;

export const showToast = (message, type = 'info') => {
  const id = idCounter++;
  toasts.value.push({ id, message, type });

  setTimeout(() => {
    removeToast(id);
  }, 2600);
};

export const toast = {
  success: (msg) => showToast(msg, 'success'),
  error: (msg) => showToast(msg, 'error'),
  warn: (msg) => showToast(msg, 'warning'),
  info: (msg) => showToast(msg, 'info'),
};

const removeToast = (id) => {
  const index = toasts.value.findIndex(t => t.id === id);
  if (index !== -1) {
    toasts.value.splice(index, 1);
  }
};
