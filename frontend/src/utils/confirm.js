import { ref } from 'vue';

export const confirmState = ref({
  isOpen: false,
  title: '',
  message: '',
  resolve: null
});

export const showConfirm = (message, title = '确认操作') => {
  return new Promise((resolve) => {
    confirmState.value = {
      isOpen: true,
      title,
      message,
      resolve
    };
  });
};

export const confirmModal = (title, message) => {
  return showConfirm(message || title, message ? title : '确认操作');
};

export const closeConfirm = (result) => {
  if (confirmState.value.resolve) {
    confirmState.value.resolve(result);
  }
  confirmState.value.isOpen = false;
};
