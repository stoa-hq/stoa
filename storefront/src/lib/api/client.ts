const API_BASE = '/api/v1';

export interface ApiResponse<T> {
	data?: T;
	meta?: { total: number; page: number; limit: number; pages: number };
	errors?: { code: string; detail: string; field?: string }[];
}

// ── CSRF ─────────────────────────────────────────────────────────────────────

function readCsrfCookie(): string | null {
	if (typeof document === 'undefined') return null;
	const m = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]+)/);
	return m ? decodeURIComponent(m[1]) : null;
}

let csrfPrimingPromise: Promise<void> | null = null;

async function ensureCsrfToken(): Promise<string | null> {
	const token = readCsrfCookie();
	if (token) return token;
	if (!csrfPrimingPromise) {
		csrfPrimingPromise = fetch(`${API_BASE}/health`)
			.catch(() => {})
			.then(() => {});
	}
	await csrfPrimingPromise;
	return readCsrfCookie();
}

// ── Auth token ───────────────────────────────────────────────────────────────

const ACCESS_TOKEN_KEY = 'storefront_access_token';
const REFRESH_TOKEN_KEY = 'storefront_refresh_token';

export function getAccessToken(): string | null {
	if (typeof localStorage === 'undefined') return null;
	return localStorage.getItem(ACCESS_TOKEN_KEY);
}

export function getRefreshToken(): string | null {
	if (typeof localStorage === 'undefined') return null;
	return localStorage.getItem(REFRESH_TOKEN_KEY);
}

export function setTokens(access: string, refresh: string) {
	if (typeof localStorage === 'undefined') return;
	localStorage.setItem(ACCESS_TOKEN_KEY, access);
	localStorage.setItem(REFRESH_TOKEN_KEY, refresh);
}

export function clearTokens() {
	if (typeof localStorage === 'undefined') return;
	localStorage.removeItem(ACCESS_TOKEN_KEY);
	localStorage.removeItem(REFRESH_TOKEN_KEY);
}

// ── Core request ─────────────────────────────────────────────────────────────

const MUTATING = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);

export async function request<T>(
	method: string,
	path: string,
	body?: unknown,
	opts: { auth?: boolean; formData?: FormData } = {}
): Promise<ApiResponse<T>> {
	const headers: Record<string, string> = {};

	const accessToken = getAccessToken();
	if (accessToken) {
		headers['Authorization'] = `Bearer ${accessToken}`;
	} else if (MUTATING.has(method)) {
		const csrf = await ensureCsrfToken();
		if (csrf) headers['X-CSRF-Token'] = csrf;
	}

	if (body !== undefined && !opts.formData) {
		headers['Content-Type'] = 'application/json';
	}

	const res = await fetch(`${API_BASE}${path}`, {
		method,
		headers,
		credentials: 'same-origin',
		body: opts.formData ?? (body !== undefined ? JSON.stringify(body) : undefined)
	});

	if (!res.ok && res.status !== 404) {
		const data = await res.json().catch(() => ({}));
		throw Object.assign(new Error(data?.errors?.[0]?.detail ?? `HTTP ${res.status}`), {
			status: res.status,
			data
		});
	}

	return res.json();
}

export const api = {
	get: <T>(path: string) => request<T>('GET', path),
	post: <T>(path: string, body: unknown) => request<T>('POST', path, body),
	put: <T>(path: string, body: unknown) => request<T>('PUT', path, body),
	delete: <T>(path: string) => request<T>('DELETE', path)
};
