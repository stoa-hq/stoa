import { writable } from 'svelte/store';

type Theme = 'dark' | 'light';

const STORAGE_KEY = 'stoa_admin_theme';

function createThemeStore() {
	const { subscribe, set } = writable<Theme>('dark');

	return {
		subscribe,
		init() {
			const stored = localStorage.getItem(STORAGE_KEY) as Theme | null;
			if (stored === 'light' || stored === 'dark') {
				set(stored);
				applyTheme(stored);
			} else if (window.matchMedia('(prefers-color-scheme: light)').matches) {
				set('light');
				applyTheme('light');
			} else {
				set('dark');
				applyTheme('dark');
			}
		},
		toggle() {
			const html = document.documentElement;
			const isDark = html.classList.contains('dark');
			const next: Theme = isDark ? 'light' : 'dark';
			set(next);
			localStorage.setItem(STORAGE_KEY, next);
			applyTheme(next);
		}
	};
}

function applyTheme(theme: Theme) {
	if (theme === 'dark') {
		document.documentElement.classList.add('dark');
	} else {
		document.documentElement.classList.remove('dark');
	}
}

export const theme = createThemeStore();
