import { writable } from 'svelte/store';

interface AuthState {
	accessToken: string | null;
	refreshToken: string | null;
	user: { email: string; role: string } | null;
}

const ACCESS_TOKEN_KEY = 'stoa_access_token';
const REFRESH_TOKEN_KEY = 'stoa_refresh_token';

function parseJwtPayload(token: string): Record<string, unknown> | null {
	try {
		// JWT uses base64url (- instead of +, _ instead of /, no padding).
		// atob() requires standard base64 with padding, so convert first.
		const b64url = token.split('.')[1];
		const b64 = b64url.replace(/-/g, '+').replace(/_/g, '/');
		const padded = b64.padEnd(b64.length + (4 - (b64.length % 4)) % 4, '=');
		return JSON.parse(atob(padded));
	} catch {
		return null;
	}
}

function createAuthStore() {
	// Restore from localStorage on init
	const initial: AuthState = {
		accessToken: null,
		refreshToken: null,
		user: null
	};

	if (typeof localStorage !== 'undefined') {
		const at = localStorage.getItem(ACCESS_TOKEN_KEY);
		const rt = localStorage.getItem(REFRESH_TOKEN_KEY);
		if (at) {
			initial.accessToken = at;
			initial.refreshToken = rt;
			const payload = parseJwtPayload(at);
			if (payload) {
				initial.user = {
					email: (payload.email as string) ?? '',
					role: (payload.role as string) ?? ''
				};
			}
		}
	}

	const { subscribe, set, update } = writable<AuthState>(initial);

	return {
		subscribe,
		setTokens(accessToken: string, refreshToken: string) {
			const payload = parseJwtPayload(accessToken);
			const user = payload
				? { email: (payload.email as string) ?? '', role: (payload.role as string) ?? '' }
				: null;

			if (typeof localStorage !== 'undefined') {
				localStorage.setItem(ACCESS_TOKEN_KEY, accessToken);
				localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
			}

			set({ accessToken, refreshToken, user });
		},
		logout() {
			if (typeof localStorage !== 'undefined') {
				localStorage.removeItem(ACCESS_TOKEN_KEY);
				localStorage.removeItem(REFRESH_TOKEN_KEY);
			}
			set({ accessToken: null, refreshToken: null, user: null });
		},
		isAuthenticated(): boolean {
			let auth: AuthState = { accessToken: null, refreshToken: null, user: null };
			const unsubscribe = this.subscribe((v) => (auth = v));
			unsubscribe();
			if (!auth.accessToken) return false;
			const payload = parseJwtPayload(auth.accessToken);
			if (!payload) return false;
			const exp = payload.exp as number;
			return Date.now() / 1000 < exp;
		}
	};
}

export const authStore = createAuthStore();
