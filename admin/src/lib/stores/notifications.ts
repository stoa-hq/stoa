import { writable } from 'svelte/store';

export type NotificationType = 'success' | 'error' | 'info' | 'warning';

export interface Notification {
	id: string;
	type: NotificationType;
	message: string;
	duration?: number;
}

function createNotificationStore() {
	const { subscribe, update } = writable<Notification[]>([]);

	function add(type: NotificationType, message: string, duration = 4000) {
		const id = Math.random().toString(36).slice(2);
		update((n) => [...n, { id, type, message, duration }]);
		if (duration > 0) {
			setTimeout(() => remove(id), duration);
		}
		return id;
	}

	function remove(id: string) {
		update((n) => n.filter((x) => x.id !== id));
	}

	return {
		subscribe,
		success: (msg: string) => add('success', msg),
		error: (msg: string) => add('error', msg, 6000),
		info: (msg: string) => add('info', msg),
		warning: (msg: string) => add('warning', msg),
		remove
	};
}

export const notifications = createNotificationStore();
