import { ref, inject, provide, type InjectionKey, type Ref } from "vue";

export interface ToastItem {
  id: number;
  message: string;
  type: "success" | "error";
}

export interface ToastContext {
  toasts: Ref<ToastItem[]>;
  toast: (
    message: string,
    type?: "success" | "error",
    duration?: number,
  ) => void;
}

let nextId = 0;

const toastKey: InjectionKey<ToastContext> = Symbol("toast");

export function provideToast(): ToastContext {
  const toasts = ref<ToastItem[]>([]);

  function toast(
    message: string,
    type: "success" | "error" = "success",
    duration = 3000,
  ) {
    const id = nextId++;
    toasts.value.push({ id, message, type });
    setTimeout(() => {
      toasts.value = toasts.value.filter((t) => t.id !== id);
    }, duration);
  }

  const ctx: ToastContext = { toasts, toast };
  provide(toastKey, ctx);
  return ctx;
}

export function useToast(): ToastContext {
  const ctx = inject(toastKey);
  if (!ctx) {
    throw new Error(
      "useToast() must be used within a component that called provideToast()",
    );
  }
  return ctx;
}
